package main

/*

This can be tested locally using a simple TCP listener.
I use the one here: https://coderwall.com/p/wohavg/creating-a-simple-tcp-server-in-go

*/

import (
	"context"
	_ "expvar"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"time"

	_ "github.com/alexbrainman/odbc"
	singleinstance "github.com/allan-simon/go-singleinstance"
	"github.com/billgraziano/xelogstash/applog"
	"github.com/billgraziano/xelogstash/config"
	"github.com/billgraziano/xelogstash/pkg/rotator"
	"github.com/billgraziano/xelogstash/summary"
	"github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

var opts struct {
	TOMLFile string `long:"config" description:"Read configuration from this TOML file"`
	Log      bool   `long:"log" description:"Also write to log file based on the EXE name"`
	Debug    bool   `long:"debug" description:"Enable debug logging"`
}

var appConfig config.App
var globalConfig config.Config

func runApp() error {

	var err error

	var parser = flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	_, err = parser.Parse()
	if err != nil {
		log.Error(errors.Wrap(err, "flags.Parse"))
		return err
	}

	if opts.Debug {
		log.SetLevel(log.DebugLevel)
	}

	// Log to file
	if opts.Log {
		logger := rotator.New("log", "xelogstash", "log")

		log.SetOutput(logger)
		log.SetFormatter(&formatter{
			fields: logrus.Fields{
				"version":     version,
				"version_git": sha1ver,
			},
			lf: &log.JSONFormatter{},
		})
		log.Debug(fmt.Sprintf("log retention: %s", logger.Retention.String()))
		err = logger.Start()
		if err != nil {
			log.Error(err)
			return errors.Wrap(err, "rotator.start")
		}
		defer logger.Close()
	}

	log.Info(fmt.Sprintf("version: %s (%s @ %s)", version, sha1ver, buildTime))
	// use default config file if one isn't specified
	var fn string
	if opts.TOMLFile == "" {
		fn, err = getDefaultConfigFileName()
		if err != nil {
			log.Error(errors.Wrap(err, "getdefaultconfigfilename"))
			return err
		}
		opts.TOMLFile = fn
	}

	var settings config.Config
	settings, err = config.Get(opts.TOMLFile, version, sha1ver)
	if err != nil {
		log.Error(errors.Wrap(err, "config.get"))
		return err
	}

	if settings.App.Workers == 0 {
		settings.App.Workers = runtime.NumCPU()
	}

	err = applog.Initialize(settings)
	if err != nil {
		log.Error(errors.Wrap(err, "applog.init"))
		return err
	}

	var logMessage string
	logMessage = fmt.Sprintf("app-start: workers %d; default rows: %d", settings.App.Workers, settings.Defaults.Rows)
	log.Info(logMessage)
	err = applog.Info(logMessage)
	if err != nil {
		log.Error("applog to logstash failed:", err)
		return err
	}

	// Report the app logstash
	if settings.AppLog.Logstash != "" {
		logMessage = fmt.Sprintf("applog.logstash: %s", settings.AppLog.Logstash)
		log.Info(logMessage)
	}

	// check the lock file
	lockFileName := opts.TOMLFile + ".lock"
	lockfile, err := singleinstance.CreateLockFile(lockFileName)
	if err != nil {
		msg := fmt.Sprintf("instance running: lock file: %s", lockFileName)
		log.Error(msg)
		applog.Error(msg)
		return err
	}

	appConfig = settings.App
	globalConfig = settings

	sinks, err := globalConfig.GetSinks()
	if err != nil {
		return errors.Wrap(err, "globalconfig.getsinks")
	}
	for i := range sinks {
		log.Info(fmt.Sprintf("Destination: %s", sinks[i].Name()))
	}

	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: http.DefaultServeMux,
	}

	// Enables a web server on :8080 with basic metrics
	if appConfig.HTTPMetrics {
		go func() {
			log.Debug("HTTP metrics server starting...")
			//http.ListenAndServe(":8080", http.DefaultServeMux)
			serverErr := httpServer.ListenAndServe()
			if serverErr == http.ErrServerClosed {
				log.Debug("HTTP metrics server closed")
				return
			}
			if serverErr != nil {
				log.Error(errors.Wrap(errors.Wrap(serverErr, "http.listenandserve"), "appconfig.httpmetrics"))
			}
			log.Debug("HTTP metrics server closed fallthrough")
		}()
	}

	message, cleanRun := processall(settings)
	log.Info(message)
	if cleanRun {
		err = applog.Info(message)
	} else {
		err = applog.Error(message)
	}
	if err != nil {
		log.Error("applog to logstash failed:", err)
	}

	if settings.App.Summary {
		log.Debug("Printing summary...")
		summary.PrintSummary()
	}

	if settings.App.Samples {
		log.Debug("Printing JSON samples...")
		err = summary.PrintSamples()
		if err != nil {
			log.Error(errors.Wrap(err, "summary.printsamples"))
		}
	}

	if settings.AppLog.Samples {
		log.Debug("Printing app log samples...")
		err = applog.PrintSamples()
		if err != nil {
			log.Error(errors.Wrap(err, "applog.samples"))
		}
	}

	log.Debug("Cleaning old log files...")
	err = cleanOldLogFiles(7)
	if err != nil {
		log.Error(errors.Wrap(err, "cleanOldLogFiles"))
	}

	err = applog.Close()
	if err != nil {
		log.Error(errors.Wrap(err, "applog.close"))
	}

	if appConfig.HTTPMetrics {
		log.Debug("HTTP metrics server stopping...")
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		httpServer.SetKeepAlivesEnabled(false)
		err = httpServer.Shutdown(ctx)
		if err != nil {
			log.Error(errors.Wrap(err, "http.shutdown"))
		}
		// give this time to close?
		time.Sleep(1 * time.Second)
	}

	log.Debug("Closing lock file...")
	err = closeLockFile(lockfile)
	if err != nil {
		msg := errors.Wrap(err, "closelockfile").Error()
		log.Error(msg)
		applog.Error(msg)
		cleanRun = false
	}

	log.Debug("Removing lock file...")
	err = removeLockFile(lockFileName)
	if err != nil {
		msg := errors.Wrap(err, "removelockfile").Error()
		log.Error(msg)
		applog.Error(msg)
		cleanRun = false
	}
	log.Debug("Returned from removing lock file...")

	if !cleanRun {
		log.Error("*** ERROR ****")
		return err
	}

	return nil
}

// used for logrus custom fields
type formatter struct {
	fields log.Fields
	lf     log.Formatter
}

// Format satisfies the logrus.Formatter interface.
func (f *formatter) Format(e *log.Entry) ([]byte, error) {
	for k, v := range f.fields {
		e.Data[k] = v
	}
	return f.lf.Format(e)
}
