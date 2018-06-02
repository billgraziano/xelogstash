package main

/*

This can be tested locally using a simple TCP listener.
I use the one here: https://coderwall.com/p/wohavg/creating-a-simple-tcp-server-in-go

*/

import (
	_ "expvar"
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"

	"github.com/billgraziano/xelogstash/applog"

	"github.com/billgraziano/xelogstash/log"
	"github.com/billgraziano/xelogstash/summary"

	_ "github.com/alexbrainman/odbc"
	"github.com/billgraziano/xelogstash/config"
	"github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
)

const version = "0.21"

var sha1ver string

var opts struct {
	TOMLFile string `long:"config" description:"Read configuration from this TOML file"`
	Log      bool   `long:"log" description:"Also write to log file based on the EXE name"`
	Debug    bool   `long:"debug" description:"Enable debug logging"`
}

var appConfig config.App

func main() {

	var err error

	// stop := profile.Start(profile.ProfilePath("."))
	// f, err := os.Create("cpu_profile.prof")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// if err = pprof.StartCPUProfile(f); err != nil {
	// 	log.Fatal("pprof.cpu.start", err)
	// }
	// defer pprof.StopCPUProfile()
	// defer f.Close()

	log.SetFlags(log.LstdFlags | log.LUTC)

	// did we get a full SHA1?
	if len(sha1ver) == 40 {
		sha1ver = sha1ver[0:7]
	}

	if sha1ver == "" {
		sha1ver = "dev"
	}

	var parser = flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	_, err = parser.Parse()
	if err != nil {
		log.Error(errors.Wrap(err, "flags.Parse"))
		os.Exit(1)
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
			os.Exit(1)
		}

		multi := io.MultiWriter(os.Stdout, lf)
		log.SetOutput(multi)
	}

	log.Info("==================================================================")

	// use default config file if one isn't specified
	if opts.TOMLFile == "" {
		fn, err := getDefaultConfigFileName()
		if err != nil {
			log.Error(errors.Wrap(err, "getdefaultconfigfilename"))
			os.Exit(1)
		}
		opts.TOMLFile = fn
	}

	var settings config.Config
	settings, err = config.Get(opts.TOMLFile, version)
	if err != nil {
		log.Error(errors.Wrap(err, "config.get"))
		os.Exit(1)
	}

	// if config.App.Debug {
	// 	log.SetLevel(log.DEBUG)
	// }

	if settings.App.Workers == 0 {
		settings.App.Workers = runtime.NumCPU() * 4
	}
	err = applog.Initialize(settings.AppLog)
	if err != nil {
		log.Error(errors.Wrap(err, "applog.init"))
		os.Exit(1)
	}

	var logMessage string
	logMessage = fmt.Sprintf("app-start version: %s; workers %d; default rows: %d", version, settings.App.Workers, settings.Defaults.Rows)
	if sha1ver != "" {
		logMessage += fmt.Sprintf("; sha1: %s", sha1ver)
	}
	log.Info(logMessage)
	err = applog.Info(logMessage)
	if err != nil {
		log.Error("applog to logstash failed:", err)
		os.Exit(1)
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

	// Report hte app logstash
	if settings.AppLog.Logstash == "" {
		logMessage = "applog.logstash is empty.  Not logging application events to logstash."
	} else {
		logMessage = fmt.Sprintf("applog.logstash: %s", settings.AppLog.Logstash)
	}
	log.Info(logMessage)

	appConfig = settings.App

	// Enables a web server on :8080 with basic metrics
	if appConfig.HTTPMetrics {
		go http.ListenAndServe(":8080", http.DefaultServeMux)
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
		log.Debug("Pringing app log samples...")
		err = applog.PrintSamples()
		if err != nil {
			log.Error(errors.Wrap(err, "applog.samples"))
		}
	}

	if !cleanRun {
		log.Error("*** ERROR ****")
		os.Exit(1)
	}

	// stop.Stop()
	// time.Sleep(time.Duration(2 * time.Second))

	err = cleanOldLogFiles(7)
	log.Debug("Cleaning old log files...")
	if err != nil {
		log.Error(errors.Wrap(err, "cleanOldLogFiles"))
		os.Exit(1)
	}

	os.Exit(0)
}
