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
	configFile, err := getConfigFileName()
	if err != nil {
		log.Error(err)
		return errors.Wrap(err, "getconfigfilename")
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Error(err)
		return errors.Wrap(err, "fsnotify.newwatcher")
	}
	//defer watcher.Close()
	log.Infof("watching configuration file: %s", configFile)
	go func() {

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Info("filewatcher event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Info("file changed: ", event.Name)
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
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Error("error:", err)
			}
		}
	}()

	err = watcher.Add(configFile)
	if err != nil {
		log.Error(err)
		return errors.Wrap(err, "watcher.add")
	}
	return nil
}
