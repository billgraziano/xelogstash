package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"

	"github.com/billgraziano/xelogstash/app"
	"github.com/billgraziano/xelogstash/sink"
	"github.com/kardianos/service"
	"github.com/shiena/ansicolor"

	log "github.com/sirupsen/logrus"
)

var (
	sha1ver   = "dev"
	version   = "dev"
	buildTime string
)

func main() {
	var err error

	// TODO get a flags package that allows / options too
	svcFlag := flag.String("service", "", "Control the system service (install|uninstall)")
	debug := flag.Bool("debug", false, "Enable debug logging")
	trace := flag.Bool("trace", false, "Enable trace logging")
	filelog := flag.Bool("log", false, "Force logging to JSON file")
	flag.Parse()

	if !service.Interactive() || *filelog {
		dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			log.Fatal(err)
		}
		dir = filepath.Join(dir, "log")
		rot := sink.NewRotator(dir, "sqlxewriter", "log")

		// I'm not sure about handling and error here
		// https://www.joeshaw.org/dont-defer-close-on-writable-files/
		defer rot.Close()

		log.SetReportCaller(true)

		log.SetFormatter(&formatter{
			fields: log.Fields{
				"version":     version,
				"version_git": sha1ver,
				"application": filepath.Base(os.Args[0]),
			},
			lf: &log.JSONFormatter{
				CallerPrettyfier: func(f *runtime.Frame) (string, string) {
					filename := path.Base(f.File)
					return fmt.Sprintf("%s()", f.Function), fmt.Sprintf("%s:%d", filename, f.Line)
				},
			},
		})
		log.SetOutput(rot)
	} else {
		// force colors on for TextFormatter
		log.SetFormatter(&log.TextFormatter{
			ForceColors: true,
		})

		log.SetOutput(ansicolor.NewAnsiColorWriter(os.Stdout))
	}

	log.Infof("start: version: %s; git: %s; build: %s", version, sha1ver, buildTime)

	if *debug {
		log.SetLevel(log.DebugLevel)
		log.Debug("log level: debug")
	}

	if *trace {
		log.SetLevel(log.TraceLevel)
		log.Trace("log level: trace")
	}

	svcConfig := &service.Config{
		Name:        "sqlxewriter",
		DisplayName: "SQL Server XE Writer Service",
		Description: "SQL Server XE Writer Service",
	}

	prg := &app.Program{
		SHA1: sha1ver,
		//Debug:        *debug,
		PollInterval: 10,
		ExtraDelay:   2,
	}
	// if *debug {
	// 	prg.Debug = true
	// }
	svc, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	if len(*svcFlag) != 0 {
		log.Infof("processing action: %s", *svcFlag)
		err = service.Control(svc, *svcFlag)
		if err != nil {
			log.Errorf("valid actions: %q\n", service.ControlAction)
			log.Fatal(err)
		}
		log.Infof("action %s: sucessful", *svcFlag)
		return
	}
	err = svc.Run()
	if err != nil {
		log.Fatal(err)
	}
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
