package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"

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
	builtBy   = "dev"
)

func main() {
	var err error

	svcFlag := flag.String("service", "", "Control the system service (install|uninstall)")
	debug := flag.Bool("debug", false, "Enable debug logging")
	trace := flag.Bool("trace", false, "Enable trace logging")
	filelog := flag.Bool("log", false, "Force logging to JSON file")
	loop := flag.Bool("loop", false, "continue polling until canceleld (command-line only)")
	versionOnly := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	appdir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}

	// if len(sha1ver) > 7 {
	// 	sha1ver = sha1ver[:7]
	// }

	// we are logging to a file
	if !service.Interactive() || *filelog {

		dir := filepath.Join(appdir, "log")
		rot := sink.NewRotator(dir, "sqlxewriter", "log")

		// I'm not sure about handling an error here
		// https://www.joeshaw.org/dont-defer-close-on-writable-files/
		defer rot.Close()

		// include the calling method as a field
		//log.SetReportCaller(true)

		// https://github.com/sirupsen/logrus/blob/master/example_custom_caller_test.go
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
				TimestampFormat: "2006-01-02T15:04:05.999999999-07:00",
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
	if *versionOnly {
		return
	}

	defer func() {
		err := recover()
		if err != nil {
			log.Error(fmt.Sprintf("panic:  %#v\n", err))
			buf := make([]byte, 4096)
			buf = buf[:runtime.Stack(buf, true)]
			log.Error(fmt.Sprintf("%s\n", buf))
		}
	}()

	prg := &app.Program{
		SHA1:       sha1ver,
		Version:    version,
		ExtraDelay: 0,
		StartTime:  time.Now(),
		LogLevel:   log.InfoLevel,
		Loop:       *loop,
		//PollInterval: 60,
	}

	// if we are running as a service or loop is true
	if !service.Interactive() || *loop {
		prg.Loop = true
	}

	if *debug {
		log.SetLevel(log.DebugLevel)
		log.Info("log level: debug")
		prg.LogLevel = log.DebugLevel
	}

	if *trace {
		log.SetLevel(log.TraceLevel)
		log.Info("log level: trace")
		prg.LogLevel = log.TraceLevel
	}

	description := fmt.Sprintf("SQL Server Extended Event Writer (%s.exe) from https://github.com/billgraziano/xelogstash", filepath.Join(appdir, os.Args[0]))
	log.Trace(description)
	svcConfig := &service.Config{
		Name:        "sqlxewriter",
		DisplayName: "XEvent Writer for SQL Server",
		Description: description,
	}

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

	log.Tracef("loop: %t", *loop)
	// if we're running as a service or we are looping...
	if prg.Loop {
		log.Tracef("starting svc.run")
		err = svc.Run()
		if err != nil {
			log.Fatal(err)
		}
		log.Tracef("svc.run exited")
	} else {
		log.Tracef("running once...")
		err = prg.Start(svc)
		if err != nil {
			log.Fatal(err)
		}
		err = prg.Stop(svc)
		if err != nil {
			log.Fatal(err)
		}
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
