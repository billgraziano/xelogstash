package app

import (
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func (p *Program) startWatching() (err error) {
	if !p.WatchConfig {
		return nil
	}
	config, sources, err := getConfigFiles()
	if err != nil {
		log.Error(err)
		return errors.Wrap(err, "getconfigfiles")
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Error(err)
		return errors.Wrap(err, "fsnotify.newwatcher")
	}
	ch := make(chan int)
	go coalesce(watcher.Events, ch)

	go func() {

		for {
			select {
			case count, ok := <-ch:
				if !ok {
					return
				}
				log.Infof("configuration file changed: events: %d", count)
				err = p.stopPolling()
				if err != nil {
					log.Error(errors.Wrap(err, "stoppolling"))
				}
				time.Sleep(1 * time.Second)
				err = p.startPolling()
				if err != nil {
					log.Error(errors.Wrap(err, "startpolling"))
				}
				time.Sleep(1 * time.Second)
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Error("error:", err)
			}
		}
	}()

	err = watcher.Add(config)
	if err != nil {
		log.Error(err)
		return errors.Wrap(err, "watcher.add")
	}
	log.Infof("watching configuration file: %s", config)

	if sources != "" {
		err = watcher.Add(sources)
		if err != nil {
			log.Error(err)
			return errors.Wrap(err, "watcher.add")
		}
		log.Infof("watching sources file: %s", sources)
	}

	return nil
}

// coalesce watches fsnotify events and returns when no new event has happened
// for one second or after five seconds
func coalesce(in <-chan fsnotify.Event, out chan<- int) {

	timer := time.NewTicker(1 * time.Second)
	var events int // count of events

	active := false
	first := time.Time{}
	last := time.Time{}

	for {
		select {
		case e := <-in:
			events++
			log.Debugf("filewatcher-in: %s:%s (%d)", e.Name, e.Op.String(), events)
			last = time.Now()
			if !active {
				first = time.Now()
			}
			active = true

		case <-timer.C:
			if active {
				if time.Since(first) > time.Duration(5*time.Second) || time.Since(last) > time.Duration(2*time.Second) {
					log.Debugf("filwatcher-out: active: %v first:%v last:%v", active, first, last)
					out <- events
					active = false
					events = 0
				}
			}
		}
	}
}
