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
	"strings"
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
		log.Trace("running interactively")
	} else {
		log.Trace("running under service manager")
	}

	ConfigureExpvar()
	http.Handle("/debug/metrics", metric.Handler(metric.Exposed))

	err = p.startPolling()
	if err != nil {
		log.Error(errors.Wrap(err, "startpolling"))
		return errors.Wrap(err, "startpolling")
	}

	err = p.startWatching()
	if err != nil {
		log.Error(errors.Wrap(err, "startwatching"))
		log.Error("reload on error probably disabled")
	}

	log.Trace("app.start exiting")
	return nil
}

func (p *Program) startPolling() (err error) {

	log.Trace("app.program.startpolling...")
	p.Lock()
	defer p.Unlock()
	ctx, cancel := context.WithCancel(context.Background())
	p.Cancel = cancel

	p.Server = nil

	// Read the config file
	settings, err := p.getConfig()
	if err != nil {
		return errors.Wrap(err, "p.getconfig")
	}
	log.Infof("config file: %s", settings.FileName)
	if settings.SourcesFile != "" {
		log.Infof("sources file: %s", settings.SourcesFile)
	}
	if len(p.Filters) > 0 {
		log.Info(fmt.Sprintf("filters: %d", len(p.Filters)))
	}
	p.WatchConfig = settings.App.WatchConfig

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

	if settings.App.HTTPMetrics {
		err = p.enableHTTP(settings.App.HTTPMetricsPort)
		if err != nil {
			log.Error(errors.Wrap(err, "enablehttp"))
		}
	}

	p.targets = len(settings.Sources)
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
	log.Tracef("loop: %t", p.Loop)
	if p.Loop {
		for i := 0; i < p.targets; i++ {
			go p.run(ctx, i, settings)
		}

		go func(ctx context.Context, count int) {
			p.logMemory(ctx, count)
		}(ctx, p.targets)

	} else {
		for i := 0; i < p.targets; i++ {
			p.run(ctx, i, settings)
		}
		writeMemory(p.StartTime, 1)
	}

	return nil
}

func (p *Program) fileSave() {

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
	if p.Loop {
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

		if !p.Loop {
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

// stopPolling stops all polling and closes all sinks
func (p *Program) stopPolling() (err error) {
	p.Lock()
	defer p.Unlock()

	log.Debug("sending cancel to pollers...")
	p.Cancel()
	p.wg.Wait()

	badClose := false
	log.Trace("closing sinks...")
	for i := range p.Sinks {
		snk := *p.Sinks[i]
		err = snk.Close()
		if err != nil {
			log.Error(errors.Wrap(err, fmt.Sprintf("close: sink: %s", snk.Name())))
			badClose = true
		}
	}
	log.Info("all sinks closed")
	if badClose {
		return errors.Wrap(err, "sink.close")
	}

	// shutdown the HTTP server, if not nil
	if p.Server != nil {
		log.Trace("http.server.shutdown...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err = p.Server.Shutdown(ctx)
		if err != nil {
			log.Error(err)
		}
		p.Server = nil
	}

	// clean out the map of servers so we can restart
	status.Reset()

	return nil
}

// Stop handles a stop request
func (p *Program) Stop(s service.Service) error {
	var err error
	log.Info("stop signal received")
	err = p.stopPolling()
	if err != nil {
		return errors.Wrap(err, "stoppolling")
	}
	log.Info("application stopping")
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

	cfg, src, err := getConfigFiles()
	if err != nil {
		return c, errors.Wrap(err, "getconfigfiles")
	}
	c, err = config.Get(cfg, src, p.Version, p.SHA1)
	if err != nil {
		return c, errors.Wrap(err, "config.get")
	}
	p.Filters = c.Filters
	return c, nil
}

func getConfigFiles() (cfg string, src string, err error) {

	// get the dir of the EXE
	// c:\dir\sqlxewriter.exe
	exe, err := os.Executable()
	if err != nil {
		return "", "", errors.Wrap(err, "os.executable")
	}
	exePath := filepath.Dir(exe)
	base := filepath.Base(exe)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	config := filepath.Join(exePath, base+".toml")
	log.Debugf("desired config file: %s", config)

	// Does the desired config file exist
	_, err = os.Stat(config)
	if os.IsNotExist(err) {
		return "", "", fmt.Errorf("missing config file: %s", config)
	}
	if err != nil {
		return "", "", errors.Wrap(err, "os.stat")
	}
	cfg = config

	// Does the sources file exist
	sources := filepath.Join(exePath, base+"_sources.toml")
	log.Debugf("checking sources file: %s", sources)
	_, err = os.Stat(sources)
	if os.IsNotExist(err) {
		return cfg, "", nil
	}
	if err != nil {
		return "", "", errors.Wrap(err, "os.state: sources")
	}
	src = sources
	return cfg, src, nil
}

func (p *Program) enableHTTP(port int) error {
	addr := fmt.Sprintf(":%d", port)

	log.Infof("pprof available at http://localhost:%d/debug/pprof", port)
	log.Infof("expvars available at http://localhost:%d/debug/vars", port)
	log.Infof("metrics available at http://localhost:%d/debug/metrics", port)

	p.Server = &http.Server{
		Addr:    addr,
		Handler: http.DefaultServeMux,
	}
	//

	go func() {
		serverErr := p.Server.ListenAndServe()
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
