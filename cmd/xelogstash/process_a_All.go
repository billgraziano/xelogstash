package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/billgraziano/xelogstash/applog"
	"github.com/pkg/errors"

	"github.com/billgraziano/xelogstash/config"
	"github.com/billgraziano/xelogstash/log"
	humanize "github.com/dustin/go-humanize"
	"github.com/dustin/go-humanize/english"
)

func processall(settings config.Config) (string, bool) {
	start := time.Now()
	cleanRun := true

	msg := fmt.Sprintf("Processing %d source(s)...", len(settings.Sources))
	log.Info(msg)
	err := applog.Info(msg)
	if err != nil {
		log.Error(errors.Wrap(err, "applog.info"))
	}

	// how big to make channels
	maxSessions := len(settings.Defaults.Sessions)
	for _, src := range settings.Sources {
		if len(src.Sessions) > maxSessions {
			maxSessions = len(src.Sessions)
		}
	}
	// the +1 is for agent jobs
	maxSize := (maxSessions + 1) * len(settings.Sources) * 2

	jobs := make(chan config.Source, maxSize)
	results := make(chan int, maxSize)
	exceptions := make(chan error, maxSize)

	var wg sync.WaitGroup
	for w := 1; w <= settings.App.Workers; w++ {
		go worker(w, &wg, jobs, results, exceptions)
	}

	for _, s := range settings.Sources {
		wg.Add(1)
		jobs <- s
	}

	close(jobs)
	wg.Wait()

	close(results)
	//close(exceptions)

	select {
	case <-exceptions:
		cleanRun = false
	default:
	}

	var rows, sources int
	for r := range results {
		sources++
		rows += r
	}

	runtime := time.Since(start)
	seconds := runtime.Seconds()
	// TODO - based on the error, generate a message

	var rowsPerSecond int64
	if seconds > 0.0 {
		rowsPerSecond = int64(float64(rows) / seconds)
	}

	var textMessage string
	if rowsPerSecond > 0 && seconds > 1 {
		textMessage = fmt.Sprintf("Processed %s %s - %s per second (%s)",
			humanize.Comma(int64(rows)),
			english.PluralWord(rows, "event", ""),
			humanize.Comma(int64(rowsPerSecond)),
			roundDuration(runtime, time.Second))
	} else {
		textMessage = fmt.Sprintf("Processed %s %s", humanize.Comma(int64(rows)), english.PluralWord(rows, "event", ""))
	}

	// TODO Send to logstash
	return textMessage, cleanRun
}
