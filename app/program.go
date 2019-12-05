package app

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"sync"
	"time"

	"github.com/billgraziano/xelogstash/config"
	"github.com/billgraziano/xelogstash/sink"
	"github.com/kardianos/service"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Program is used to launch the servcie
type Program struct {
	SHA1    string
	Version string

	//Debug bool

	wg      sync.WaitGroup
	cancel  context.CancelFunc
	targets int

	// Interval that we poll servers in seconds
	PollInterval int

	// ExtraDelay that is added to for testing
	// the stop function (in seconds)
	// This will probably be removed.
	ExtraDelay int

	// If we're writing to a file sink, which is an event rotator,
	// this has a pointer to it so we can close it at the very end
	eventRotator *sink.Rotator
}

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
	p.cancel = cancel

	// Read the config file
	var settings config.Config
	settings, err = config.Get("sqlxewriter.toml", p.Version, p.SHA1)
	if err != nil {
		log.Error(errors.Wrap(err, "config.get"))
		return err
	}

	if settings.App.LogLevel != "" {
		lvl, err := log.ParseLevel(settings.App.LogLevel)
		if err != nil {
			log.Error(errors.Wrap(err, "error parsing log level"))
		} else {
			log.Infof("log level: %v", lvl)
			log.SetLevel(lvl)
		}
	}

	msg := fmt.Sprintf("poll interval: %ds", p.PollInterval)
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
	for i := range sinks {
		log.Info(fmt.Sprintf("destination: %s", sinks[i].Name()))
	}
	p.eventRotator = settings.GetRotator()

	// TODO Enable the http server
	// TODO Enable the PPROF server
	go func() {
		log.Info(http.ListenAndServe("localhost:6060", nil))
	}()

	// launch the polling go routines
	for i := 0; i < p.targets; i++ {
		go p.run(ctx, i, settings)
	}

	return nil
}

func (p *Program) run(ctx context.Context, id int, cfg config.Config) {

	counter := 1
	log.Infof("[%d] goroutine launched %v", id, service.Platform())

	p.wg.Add(1)
	defer p.wg.Done()

	// sleep to spread out the launch (ms)
	delay := p.PollInterval * 1000 * id / p.targets

	select {
	case <-ctx.Done():
		return
	case <-time.After(time.Duration(delay) * time.Millisecond):
		break
	}

	ticker := time.NewTicker(time.Duration(p.PollInterval) * time.Second)

	for {
		// run at time zero
		log.Debugf("[%d] polling at %v (#%d)...", id, time.Now(), counter)
		select {
		case <-ticker.C:
			counter++
			continue
		case <-ctx.Done():
			log.Debugf("[%d] ctx.pause at %v...", id, time.Now())
			ticker.Stop()

			// simulate a slow stop
			if p.ExtraDelay > 0 {
				<-time.After(time.Millisecond * time.Duration(rand.Intn(p.ExtraDelay*1000)))
			}
			log.Debugf("[%d] ctx.done at %v...done", id, time.Now())
			return
		}
	}
}

// Stop handles a stop request
func (p *Program) Stop(s service.Service) error {
	var err error
	log.Info("app.program.stop")
	p.cancel()
	p.wg.Wait()

	if p.eventRotator != nil {
		log.Trace("closing event rotator...")
		err = p.eventRotator.Close()
		if err != nil {
			log.Error(errors.Wrap(err, "p.eventrotator.close"))
		}
	}

	log.Info("app.program.stop...done")
	return nil
}
