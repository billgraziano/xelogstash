package app

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/billgraziano/xelogstash/pkg/metric"
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
			// only change to a lower level of logging
			if lvl > p.LogLevel {
				log.Infof("log level: %v", lvl)
				log.SetLevel(lvl)
			}
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

	ConfigureExpvar()

	if settings.App.HTTPMetrics {
		err = enableHTTP(settings.App.HTTPMetricsPort)
		if err != nil {
			log.Error(errors.Wrap(err, "enablehttp"))
		}
	}

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
		ptr.SetLogger(log.WithFields(log.Fields{}))
		log.Info(fmt.Sprintf("sink: %s", ptr.Name()))
		err = ptr.Open(ctx, "id")
		if err != nil {
			return errors.Wrap(err, "ptr.open")
		}
	}

	if settings.App.Verbose {
		log.Info("verbose: true")
		p.Verbose = settings.App.Verbose
	}

	// launch the polling go routines
	if p.Once {
		for i := 0; i < p.targets; i++ {
			p.run(ctx, i, settings)
		}
		writeMemory(p.StartTime, 1)
	} else {
		for i := 0; i < p.targets; i++ {
			go p.run(ctx, i, settings)
		}

		go func(ctx context.Context, count int) {
			p.logMemory(ctx, count)
		}(ctx, p.targets)
	}

	return nil
}

func (p *Program) run(ctx context.Context, id int, cfg config.Config) {

	defer func() {
		err := recover()
		if err != nil {
			log.Error(fmt.Sprintf("panic:  %#v\n", err))
			buf := make([]byte, 4096)
			buf = buf[:runtime.Stack(buf, true)]
			log.Error(fmt.Sprintf("%s\n", buf))
		}
	}()

	// get the source
	if id >= len(cfg.Sources) {
		log.Errorf("poll exiting: id: %d len(sources): %d", id, len(cfg.Sources))
		return
	}
	src := cfg.Sources[id]
	contextLogger := log.WithFields(log.Fields{
		"source": src.FQDN,
	})

	p.wg.Add(1)
	defer p.wg.Done()

	counter := 1
	contextLogger.Tracef("source: %s; poll routine launched: %v", src.FQDN, service.Platform())

	// sleep to spread out the launch (ms)
	if !p.Once {
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
	}

	logmsg := fmt.Sprintf("polling: %s; interval: %ds", src.FQDN, src.PollSeconds)
	contextLogger.Info(logmsg)

	// ok is false if duplicate or context times out
	ok := p.checkdupes(ctx, src)
	if !ok {
		return
	}

	ticker := time.NewTicker(time.Duration(src.PollSeconds) * time.Second)
	for {
		// run at time zero
		contextLogger.Tracef("source: %s; polling (#%d)...", src.FQDN, counter)
		result, err := p.ProcessSource(ctx, id, src)
		if err != nil {
			if errors.Cause(err) == xe.ErrNotFound || errors.Cause(err) == xe.ErrNoFileTarget {
				if cfg.App.StrictSessions {
					contextLogger.Errorf("instance: %s; session: %s; err: %s", result.Instance, result.Session, err)
				}
			} else {
				contextLogger.Errorf("instance: %s; session: %s; err: %s", result.Instance, result.Session, err)
			}
		}

		if p.Once {
			contextLogger.Tracef("source: %s; one poll finished", src.FQDN)
			return
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
			contextLogger.Debugf("source: %s 'program.run' received ctx.done", src.FQDN)
			return
		}
	}
}

// Stop handles a stop request
func (p *Program) Stop(s service.Service) error {
	var err error
	log.Info("stop signal received")
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

	log.Info("all sinks closed.  application stopping.")
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

func enableHTTP(port int) error {
	addr := fmt.Sprintf(":%d", port)

	log.Infof("pprof available at http://localhost:%d/debug/pprof", port)
	log.Infof("expvars available at http://localhost:%d/debug/vars", port)
	log.Infof("metrics available at http://localhost:%d/debug/metrics", port)
	http.Handle("/debug/metrics", metric.Handler(metric.Exposed))
	httpServer := &http.Server{
		Addr:    addr,
		Handler: http.DefaultServeMux,
	}

	go func() {
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
	return nil
}
