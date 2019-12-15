package main

import (
	"context"
	"fmt"
	"time"

	"github.com/billgraziano/xelogstash/app"
	"github.com/billgraziano/xelogstash/config"
	"github.com/billgraziano/xelogstash/pkg/format"
	"github.com/billgraziano/xelogstash/sink"
	humanize "github.com/dustin/go-humanize"
	"github.com/dustin/go-humanize/english"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func processall(settings config.Config) (string, bool) {
	start := time.Now()
	cleanRun := true

	msg := fmt.Sprintf("Processing %d source(s)...", len(settings.Sources))
	log.Info(msg)

	// how big to make channels
	maxSessions := len(settings.Defaults.Sessions)
	for _, src := range settings.Sources {
		if len(src.Sessions) > maxSessions {
			maxSessions = len(src.Sessions)
		}
	}
	// the +1 is for agent jobs
	//maxSize := (maxSessions + 1) * len(settings.Sources) * 2

	var rows, sources int
	// for r := range results {
	// 	sources++
	// 	rows += r
	// }

	// Process sequentially
	// make app.Program
	pgm := app.Program{}
	ctx := context.TODO()

	sinks, err := settings.GetSinks()
	if err != nil {
		return errors.Wrap(err, "settings.getsink").Error(), false
	}
	pgm.Sinks = make([]*sink.Sinker, 0)
	for i := range sinks {
		pgm.Sinks = append(pgm.Sinks, &sinks[i])
	}
	for i := range pgm.Sinks {
		ptr := *pgm.Sinks[i]
		err := ptr.Open(ctx, "id")
		if err != nil {
			return errors.Wrap(err, "ptr.open").Error(), false
		}
		defer ptr.Close()
	}

	for _, src := range settings.Sources {
		result, err := pgm.ProcessSource(ctx, 0, src)
		if err != nil {
			cleanRun = false
		}
		rows += result.Rows
		sources++
	}

	// jobs := make(chan config.Source, maxSize)
	// results := make(chan int, maxSize)
	// exceptions := make(chan error, maxSize)

	// var wg sync.WaitGroup
	// for w := 1; w <= settings.App.Workers; w++ {
	// 	go worker(w, &wg, jobs, results, exceptions)
	// }

	// for _, s := range settings.Sources {
	// 	wg.Add(1)
	// 	jobs <- s
	// }

	// close(jobs)
	// wg.Wait()

	// close(results)
	// //close(exceptions)

	// select {
	// case <-exceptions:
	// 	cleanRun = false
	// default:
	// }

	// var rows, sources int
	// for r := range results {
	// 	sources++
	// 	rows += r
	// }

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
			format.RoundDuration(runtime, time.Second))
	} else {
		textMessage = fmt.Sprintf("Processed %s %s", humanize.Comma(int64(rows)), english.PluralWord(rows, "event", ""))
	}

	// TODO Send to logstash
	return textMessage, cleanRun
}
