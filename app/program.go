package app

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	_ "net/http/pprof" // pprof
	"runtime"
	"time"

	"github.com/billgraziano/mssqlh"
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

// write a panic to time stamped file
// func writePanic(msg string, buf []byte) error {
// 	ts := time.Now().Format("20060102-150405.999")
// 	file := fmt.Sprintf("panic-%s.log", ts)
// 	header := []byte(fmt.Sprintf("panic: %s\n\n", msg))
// 	body := append(header, buf...)
// 	return ioutil.WriteFile(file, body, 0700)
// }

func (p *Program) run(ctx context.Context, id int, cfg config.Config) {

	defer func() {
		err := recover()
		if err != nil {
			msg := fmt.Sprintf("%#v", err)
			log.Error(fmt.Sprintf("panic:  %s\n", msg))
			buf := make([]byte, 4096)
			buf = buf[:runtime.Stack(buf, true)]
			log.Error(fmt.Sprintf("%s\n", buf))
			// result := writePanic(msg, buf)
			// if result != nil {
			// 	log.Error(errors.Wrap(result, "writepanic"))
			// }
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
	contextLogger.Tracef("poll routine launched: %v", service.Platform())

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

	if src.ServerNameOverride != "" || src.DomainNameOverride != "" {
		logmsg := fmt.Sprintf("%s: source_name_override: '%s' domain_name_override: '%s'", src.FQDN, src.ServerNameOverride, src.DomainNameOverride)
		contextLogger.Info(logmsg)
	}

	logmsg := fmt.Sprintf("%s: polling interval: %ds", src.FQDN, src.PollSeconds)
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
			errmsg := ""
			if result.Instance != "" {
				errmsg += fmt.Sprintf("instance: %s;", result.Instance)
			} else {
				errmsg += fmt.Sprintf("fqdn: %s;", src.FQDN)
			}

			if result.Session != "" {
				if errmsg != "" {
					errmsg += " "
				}
				errmsg += fmt.Sprintf("session: %s;", result.Session)
			}
			if errmsg != "" {
				errmsg += " "
			}
			errmsg += fmt.Sprintf("err: %s", err)
			if errors.Cause(err) == xe.ErrNotFound || errors.Cause(err) == xe.ErrNoFileTarget {
				if cfg.App.StrictSessions {
					contextLogger.Error(errmsg)
				}
			} else {
				contextLogger.Error(errmsg)
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
				// #nosec G404
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
	for {
		// TODO need a version of this with context
		cxn := mssqlh.NewConnection(src.FQDN, src.User, src.Password, "master", "sqlxewriter.exe")
		if src.Driver != "" {
			cxn.Driver = src.Driver
		}
		if src.ODBCDriver != "" {
			cxn.ODBCDriver = src.ODBCDriver
		}
		info, err := xe.GetSQLInfo(cxn.Driver, cxn.String(), src.ServerNameOverride, src.DomainNameOverride)
		if err != nil {
			// if there was an error the server could be down
			// or entered incorrectly
			// we just keep logging the error
			err = errors.Wrap(err, fmt.Sprintf("checkdupes: fqdn: %s", src.FQDN))
			log.Error(err)
			if info.DB != nil {
				err := info.DB.Close()
				if err != nil {
					log.Error(errors.Wrap(err, fmt.Sprintf("checkdupes: close: fqdn: %s", src.FQDN)))
				}
			}
		} else { // we connected and got info
			err = info.DB.Close()
			if err != nil {
				log.Error(errors.Wrap(err, fmt.Sprintf("checkdupes: close: fqdn: %s", src.FQDN)))
			}
			err = status.CheckDupeInstance(info.Domain, info.Server)
			if err == nil {
				return true // no dupe
			}
			log.Error(errors.Wrap(err, fmt.Sprintf("skipping duplicate: fqdn: '%s'; domain: '%s'; server: '%s'", src.FQDN, info.Domain, info.Server)))
			return false
		}

		// we just keep checking in case the server was down
		ticker := time.NewTicker(time.Duration(src.PollSeconds) * time.Second)
		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			ticker.Stop()
			return false
		}
	}
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
