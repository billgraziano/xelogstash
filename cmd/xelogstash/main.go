package main

/*

This can be tested locally using a simple TCP listener.
I used the one here: https://coderwall.com/p/wohavg/creating-a-simple-tcp-server-in-go

*/

import (
	"fmt"
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

const version = "0.14"

var sha1ver string

var opts struct {
	TOMLFile string `long:"config" description:"Read configuration from this TOML file."`
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

	log.SetFlags(log.Ltime)
	log.Println("Starting...")
	if sha1ver != "" {
		log.Println("repository sha1:", sha1ver[0:6])
	}

	var parser = flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	_, err = parser.Parse()
	if err != nil {
		log.Println(errors.Wrap(err, "flags.Parse"))
		os.Exit(1)
	}

	// u, err := uuid.NewV4()
	// if err != nil {
	// 	log.Println("unable to generate uuid execution_id")
	// 	os.Exit(1)
	// }

	if opts.TOMLFile == "" {
		opts.TOMLFile = "xelogstash.toml"
	}

	var settings config.Config
	settings, err = config.Get(opts.TOMLFile, version)
	if err != nil {
		log.Println(errors.Wrap(err, "config.get"))
		os.Exit(1)
	}

	// if settings.App.Debug {
	// 	opts.Debug = true
	// }

	if settings.App.Workers == 0 {
		settings.App.Workers = runtime.NumCPU()
	}
	err = applog.Initialize(settings.AppLog)
	if err != nil {
		log.Println(errors.Wrap(err, "applog.init"))
		os.Exit(1)
	}

	var logMessage string
	logMessage = fmt.Sprintf("app-start version: %s; workers %d; default rows: %d", version, settings.App.Workers, settings.Defaults.Rows)
	if sha1ver != "" {
		logMessage += fmt.Sprintf("; sha1: %s", sha1ver[0:6])
	}
	log.Println(logMessage)
	err = applog.Info(logMessage)
	if err != nil {
		log.Println("applog to logstash failed:", err)
		os.Exit(1)
	}

	// Log the logstash setting
	if settings.App.Logstash == "" {
		logMessage = "app.logstash is empty.  Not logging SQL Server events to logstash."
	} else {
		logMessage = fmt.Sprintf("app.logstash: %s", settings.App.Logstash)
	}
	log.Println(logMessage)
	_ = applog.Info(logMessage)

	// Report hte app logstash
	if settings.AppLog.Logstash == "" {
		logMessage = "applog.logstash is empty.  Not logging application events to logstash."
	} else {
		logMessage = fmt.Sprintf("applog.logstash: %s", settings.AppLog.Logstash)
	}
	log.Println(logMessage)

	appConfig = settings.App

	message, cleanRun := processall(settings)
	log.Println(message)
	if cleanRun {
		err = applog.Info(message)
	} else {
		err = applog.Error(message)
	}
	if err != nil {
		log.Println("applog to logstash failed:", err)
	}

	if settings.App.Summary {
		summary.PrintSummary()
	}

	if settings.App.Samples {
		err = summary.PrintSamples()
		if err != nil {
			log.Println(errors.Wrap(err, "summary.printsamples"))
		}
	}

	if settings.AppLog.Samples {
		err = applog.PrintSamples()
		if err != nil {
			log.Println(errors.Wrap(err, "applog.samples"))
		}
	}

	if !cleanRun {
		log.Printf("*** ERROR ****\r\n")
		os.Exit(1)
	}

	// stop.Stop()
	// time.Sleep(time.Duration(2 * time.Second))

	os.Exit(0)
}
