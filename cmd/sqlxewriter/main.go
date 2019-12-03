package main

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/billgraziano/xelogstash/app"
	"github.com/billgraziano/xelogstash/pkg/rotator"
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

	svcFlag := flag.String("service", "", "Control the system service (install|uninstall)")
	debug := flag.Bool("debug", false, "Enable debug logging")
	filelog := flag.Bool("log", false, "Force logging to JSON file")
	flag.Parse()

	if !service.Interactive() || *filelog {
		dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			log.Fatal(err)
		}
		dir = filepath.Join(dir, "log")
		rot := rotator.New(dir, "sqlxewriter", "log")
		defer rot.Close()

		log.SetFormatter(&formatter{
			fields: log.Fields{
				"version":     version,
				"version_git": sha1ver,
			},
			lf: &log.JSONFormatter{},
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
		log.Debug("debug logging enabled")
	}

	svcConfig := &service.Config{
		Name:        "sqlxewriter",
		DisplayName: "SQL Server XE Writer Service",
		Description: "SQL Server XE Writer Service",
	}

	prg := &app.Program{SHA1: sha1ver, Debug: *debug}
	prg.PollInterval = 60
	prg.ExtraDelay = 0
	if *debug {
		prg.Debug = true
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
