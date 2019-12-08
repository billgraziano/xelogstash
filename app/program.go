package app

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"time"

	"github.com/billgraziano/xelogstash/sink"
	"github.com/billgraziano/xelogstash/status"
	"github.com/billgraziano/xelogstash/xe"

	"github.com/billgraziano/xelogstash/config"
	"github.com/kardianos/service"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Start the service
func (p *Program) Start(svc service.Service) error {
	var err error
	log.Debug("app.program.start")
	if service.Interactive() {
		log.Debug("running interactively")
	} else {
		log.Debug("running under service manager")
	}

	ctx, cancel := context.WithCancel(context.Background())
	p.Cancel = cancel

	// Read the config file
	settings, err := p.getConfig()
	if err != nil {
		return errors.Wrap(err, "p.getconfig")
	}
	log.Infof("config file: %s", settings.FileName)

	if settings.App.LogLevel != "" {
		lvl, err := log.ParseLevel(settings.App.LogLevel)
		if err != nil {
			log.Error(errors.Wrap(err, "error parsing log level"))
		} else {
			log.Infof("log level: %v", lvl)
			log.SetLevel(lvl)
		}
	}

	msg := fmt.Sprintf("default poll interval: %ds", settings.Defaults.PollSeconds)
	if p.ExtraDelay > 0 {
		msg += fmt.Sprintf("; extra delay: %ds", p.ExtraDelay)
	}

	log.Info(msg)

	p.targets = len(settings.Sources)
	// p.targets = 120
	log.Infof("sources: %d; default rows: %d", p.targets, settings.Defaults.Rows)

	sinks, err := settings.GetSinks()
	if err != nil {
		return errors.Wrap(err, "globalconfig.getsinks")
	}
	p.Sinks = make([]*sink.Sinker, 0)
	for i := range sinks {
		p.Sinks = append(p.Sinks, &sinks[i])
	}
	for i := range p.Sinks {
		ptr := *p.Sinks[i]
		log.Info(fmt.Sprintf("sink: %s", ptr.Name()))
		err = ptr.Open("id")
		if err != nil {
			return errors.Wrap(err, "ptr.open")
		}
	}

	// TODO Enable the http server
	// Enable the PPROF server
	go func() {
		log.Info(http.ListenAndServe("localhost:6060", nil))
	}()

	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: http.DefaultServeMux,
	}

	if settings.App.HTTPMetrics {
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

	// launch the polling go routines
	for i := 0; i < p.targets; i++ {
		go p.run(ctx, i, settings)
	}

	return nil
}

func (p *Program) run(ctx context.Context, id int, cfg config.Config) {

	p.wg.Add(1)
	defer p.wg.Done()

	counter := 1
	log.Tracef("[%d] poll routine launched: %v", id, service.Platform())

	// get the source
	if id >= len(cfg.Sources) {
		log.Errorf("poll exiting: id: %d len(sources): %d", id, len(cfg.Sources))
		return
	}
	src := cfg.Sources[id]

	// sleep to spread out the launch (ms)
	delay := cfg.Defaults.PollSeconds * 1000 * id / p.targets
	if delay == 0 {
		delay = id
	}

	select {
	case <-ctx.Done():
		return
	case <-time.After(time.Duration(delay) * time.Millisecond):
		break
	}

	logmsg := fmt.Sprintf("polling: %s; interval: %ds", src.FQDN, src.PollSeconds)
	log.Info(logmsg)

	// ok is false if duplicate or context times out
	ok := p.checkdupes(ctx, src)
	if !ok {
		return
	}

	ticker := time.NewTicker(time.Duration(src.PollSeconds) * time.Second)
	for {
		// run at time zero
		log.Debugf("[%d] polling (#%d)...", id, counter)
		result, err := p.ProcessSource(ctx, id, src)
		if err != nil {
			if errors.Cause(err) == xe.ErrNotFound || errors.Cause(err) == xe.ErrNoFileTarget {
				if cfg.App.StrictSessions {
					log.Errorf("instance: %s; session: %s; err: %s", result.Instance, result.Session, err)
				}
			} else {
				log.Errorf("instance: %s; session: %s; err: %s", result.Instance, result.Session, err)
			}
		}

		select {
		case <-ticker.C:
			counter++
			continue
		case <-ctx.Done():
			//log.Debugf("[%d] ctx.pause at %v...", id, time.Now())
			ticker.Stop()

			// simulate a slow stop
			if p.ExtraDelay > 0 {
				<-time.After(time.Millisecond * time.Duration(rand.Intn(p.ExtraDelay*1000)))
			}
			log.Debugf("[%d] ctx.done", id)
			return
		}
	}
}

// Stop handles a stop request
func (p *Program) Stop(s service.Service) error {
	var err error
	log.Info("app.program.stop")
	p.Cancel()
	p.wg.Wait()

	log.Trace("closing sinks...")
	for i := range p.Sinks {
		snk := *p.Sinks[i]
		err = snk.Close()
		if err != nil {
			log.Error(errors.Wrap(err, fmt.Sprintf("close: sink: %s", snk.Name())))
		}
	}

	log.Info("app.program.stop...done")
	return nil
}

func (p *Program) checkdupes(ctx context.Context, src config.Source) bool {
	// try to connect in a loop until we do
	ticker := time.NewTicker(time.Duration(src.PollSeconds) * time.Second)

	for {
		// TODO need a version of this with context
		info, err := xe.GetSQLInfo(src.FQDN)
		if err != nil {
			// if there was an error the server could be down
			// or entered incorrectly
			// we just keep logging the error
			log.Error(err, fmt.Sprintf("uneachable: fqdn: %s", src.FQDN))
		} else { // we connected and got info
			err = status.CheckDupeInstance(info.Domain, info.Server)
			if err == nil {
				return true // no dupe
			}
			log.Error(errors.Wrap(err, fmt.Sprintf("skipping duplicate: fqdn: %s; domain: %s; server: %s", src.FQDN, info.Domain, info.Server)))
			return false
		}

		// we just keep checking every minute in case the server was down
		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			ticker.Stop()
			return false
		}
	}
}

func (p *Program) getConfig() (config.Config, error) {
	var c config.Config
	var err error

	// get the dir of the EXE
	exe, err := os.Executable()
	if err != nil {
		return c, errors.Wrap(err, "os.executable")
	}
	exePath := filepath.Dir(exe)

	configFiles := []string{"sqlxewriter.toml", "xelogstash.toml"}
	for _, s := range configFiles {
		fqfile := filepath.Join(exePath, s)
		_, err := os.Stat(fqfile)
		if os.IsNotExist(err) {
			continue
		}
		c, err = config.Get(fqfile, p.Version, p.SHA1)
		if err != nil {
			return c, errors.Wrap(err, fmt.Sprintf("config.get (%s)", s))
		}
		return c, nil
	}
	return c, errors.New("missing sqlxewriter.toml or xelogstash.toml")
}
