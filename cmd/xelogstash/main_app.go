package main

/*

This can be tested locally using a simple TCP listener.
I use the one here: https://coderwall.com/p/wohavg/creating-a-simple-tcp-server-in-go

*/

import (
	"context"
	_ "expvar"
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"time"

	singleinstance "github.com/allan-simon/go-singleinstance"

	"github.com/billgraziano/xelogstash/applog"

	"github.com/billgraziano/xelogstash/log"
	"github.com/billgraziano/xelogstash/summary"

	_ "github.com/alexbrainman/odbc"
	"github.com/billgraziano/xelogstash/config"
	"github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
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

	log.SetFlags(log.LstdFlags | log.LUTC)
	// log.Info(fmt.Sprintf("build: %s (git: %s) @ %s", Version, GitSummary, BuildDate))
	log.Info(fmt.Sprintf("build: %s (git: %s) @ %s", Version, sha1ver, buildTime))
	var parser = flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	_, err = parser.Parse()
	if err != nil {
		log.Error(errors.Wrap(err, "flags.Parse"))
		return err
	}

	if opts.Debug {
		log.SetLevel(log.DEBUG)
	}

	// Log to file
	if opts.Log {
		var logfilename string
		logfilename, err = getLogFileName()
		if err != nil {
			log.Error("getLogFileName", err)
		}

		var lf *os.File
		lf, err = os.OpenFile(logfilename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Error("failed to open log file")
			return err
		}

		multi := io.MultiWriter(os.Stdout, lf)
		log.SetOutput(multi)
	}

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
	settings, err = config.Get(opts.TOMLFile, Version)
	if err != nil {
		log.Error(errors.Wrap(err, "config.get"))
		return err
	}

	if settings.App.Workers == 0 {
		settings.App.Workers = runtime.NumCPU() * 4
	}

	err = applog.Initialize(settings)
	if err != nil {
		log.Error(errors.Wrap(err, "applog.init"))
		return err
	}

	var logMessage string
	logMessage = fmt.Sprintf("app-start version: %s; workers %d; default rows: %d", Version, settings.App.Workers, settings.Defaults.Rows)
	// if GitSummary != "" {
	// 	logMessage += fmt.Sprintf(" (%s @ %s)", GitSummary, BuildDate)
	// }
	log.Info(logMessage)
	err = applog.Info(logMessage)
	if err != nil {
		log.Error("applog to logstash failed:", err)
		return err
	}

	// Log the logstash setting
	if settings.App.Logstash == "" {
		logMessage = "app.logstash is empty.  Not logging SQL Server events to logstash."
	} else {
		logMessage = fmt.Sprintf("app.logstash: %s", settings.App.Logstash)
	}
	log.Info(logMessage)
	err = applog.Info(logMessage)
	if err != nil {
		log.Error("applog.info:", err)
	}

	// Report the app logstash
	if settings.AppLog.Logstash == "" {
		logMessage = "applog.logstash is empty.  Not logging application events to logstash."
	} else {
		logMessage = fmt.Sprintf("applog.logstash: %s", settings.AppLog.Logstash)
	}
	log.Info(logMessage)

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

	// If we're using elastic directly, do the index maintenance
	// if len(globalConfig.Elastic.Addresses) > 0 {
	// 	logMessage = fmt.Sprintf("elastic.addresses: %s", strings.Join(globalConfig.Elastic.Addresses, ", "))
	// 	log.Info(logMessage)
	// 	if globalConfig.Elastic.Username == "" || globalConfig.Elastic.Password == "" {
	// 		return errors.New("elastic search is missing the username or password")
	// 	}

	// 	// Set up the elastic indexes
	// 	// if globalConfig.Elastic.AutoCreateIndexes {
	// 	// 	esIndexes := make([]string, 0)
	// 	// 	if globalConfig.Elastic.DefaultIndex != "" {
	// 	// 		esIndexes = append(esIndexes, globalConfig.Elastic.DefaultIndex)
	// 	// 	}
	// 	// 	if globalConfig.Elastic.AppLogIndex != "" {
	// 	// 		esIndexes = append(esIndexes, globalConfig.Elastic.AppLogIndex)
	// 	// 	}
	// 	// 	for _, ix := range globalConfig.Elastic.EventIndexMap {
	// 	// 		esIndexes = append(esIndexes, ix)
	// 	// 	}

	// 	// 	var esClient *elasticsearch.Client
	// 	// 	esClient, err = eshelper.NewClient(globalConfig.Elastic.Addresses, globalConfig.Elastic.ProxyServer, globalConfig.Elastic.Username, globalConfig.Elastic.Password)
	// 	// 	if err != nil {
	// 	// 		return errors.Wrap(err, "eshelper.newclient")
	// 	// 	}
	// 	// 	err = eshelper.CreateIndexes(esClient, esIndexes)
	// 	// 	if err != nil {
	// 	// 		return errors.Wrap(err, "eshelper.createindexes")
	// 	// 	}
	// 	// }
	// }
	// globalConfig.Elastic.Print()
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
				log.Error(errors.Wrap(serverErr, "http.listenandserve"))
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

	// log.Debug("Cleaning old sink artifacts...")
	// for i := range globalConfig.Sinks {
	// 	err = globalConfig.Sinks[i].Clean()
	// 	if err != nil {
	// 		log.Error(errors.Wrapf(err, "sink.clean: %s", globalConfig.Sinks[i].Name()))
	// 	}
	// }

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
