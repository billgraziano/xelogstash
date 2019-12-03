package main

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/kardianos/service"
	log "github.com/sirupsen/logrus"
)

// Program is used to launch the servcie
type Program struct {
	SHA1   string
	Debug  bool
	wg     sync.WaitGroup
	cancel context.CancelFunc
	count  int
}

var (
	// how often to we run a task (milliseconds)
	cycle = 10000
	// extra delay to add for testing (milliseconds)
	extraDelay = 5000
)

// Start the service
func (p *Program) Start(svc service.Service) error {
	log.Debug("app.program.start")
	if service.Interactive() {
		log.Debug("running interactively")
	} else {
		log.Debug("running under service manager")
	}

	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	p.count = 7
	for i := 0; i < p.count; i++ {
		go p.run(ctx, i)
	}

	return nil
}

func (p *Program) run(ctx context.Context, id int) {

	counter := 1
	log.Infof("[%d] goroutine launched %v", id, service.Platform())

	p.wg.Add(1)
	defer p.wg.Done()

	// sleep to spread out the launch
	delay := cycle * id / p.count

	select {
	case <-ctx.Done():
		return
	case <-time.After(time.Duration(delay) * time.Millisecond):
		break
	}

	ticker := time.NewTicker(time.Duration(cycle) * time.Millisecond)

	for {
		// run at time zero
		log.Debugf("[%d] Executing at %v (#%d)...", id, time.Now(), counter)
		select {
		case <-ticker.C:
			counter++
			continue
		case <-ctx.Done():
			log.Debugf("[%d] ctx.done at %v...pausing...", id, time.Now())
			ticker.Stop()

			// simulate a slow stop
			<-time.After(time.Millisecond * time.Duration(rand.Intn(extraDelay)))
			log.Debugf("[%d] ctx.done at %v...done", id, time.Now())
			return
		}
	}
}

// Stop handles a stop request
func (p *Program) Stop(s service.Service) error {
	log.Info("app.program.stop")
	p.cancel()
	p.wg.Wait()
	log.Info("app.program.stop...done")
	return nil
}
