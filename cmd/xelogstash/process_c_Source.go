package main

import (
	"fmt"
	"time"

	"github.com/billgraziano/xelogstash/applog"
	"github.com/billgraziano/xelogstash/config"
	"github.com/billgraziano/xelogstash/log"
	"github.com/billgraziano/xelogstash/status"
	"github.com/billgraziano/xelogstash/xe"
	humanize "github.com/dustin/go-humanize"
	"github.com/dustin/go-humanize/english"
	"github.com/pkg/errors"
)

func processSource(wid int, source config.Source) (sourceResult Result, err error) {

	var textMessage string
	info, err := xe.GetSQLInfo(source.FQDN)
	if err != nil {
		textMessage = fmt.Sprintf("[%d] %s - fqdn: %s err: %v", wid, source.Prefix, source.FQDN, err)
		log.Println(textMessage)
		_ = applog.Error(textMessage)
		return sourceResult, errors.Wrap(err, "xe.getsqlinfo")
	}

	defer safeClose(info.DB, &err)

	cleanRun := true
	for i := range source.Sessions {
		start := time.Now()
		result, err := processSession(wid, info, source, i)
		runtime := time.Since(start)
		totalSeconds := runtime.Seconds()

		var rowsPerSecond int64
		if totalSeconds > 0.0 {
			rowsPerSecond = int64(float64(result.Rows) / totalSeconds)
		}
		sourceResult.Instance = result.Instance
		sourceResult.Rows += result.Rows

		txtDuration := fmt.Sprintf(" (%s)", roundDuration(runtime, time.Second))
		if runtime.Seconds() < 10 {
			txtDuration = ""
		}

		if errors.Cause(err) == xe.ErrNotFound {
			// TODO: if strict, then warning:
			// textMessage = fmt.Sprintf("[%d] %s - %s - %s is not available.  Skipped.", wid, source.Prefix, result.Instance, result.Session)
			// _ = applog.Warn(textMessage)
			// else
			continue
		} else if err != nil {
			textMessage = fmt.Sprintf("[%d] *** ERROR: Prefix: %s - FQDN: %s - %s - %s : %s\r\n", wid, source.Prefix, source.FQDN, status.ClassXE, source.Sessions[i], err.Error())
			cleanRun = false
			_ = applog.Error(textMessage)
		} else {
			if rowsPerSecond > 0 && totalSeconds > 1 {
				textMessage = fmt.Sprintf("[%d] %s - %s - %s processed %s %s - %s per second%s",
					wid, source.Prefix, result.Instance, result.Session,
					humanize.Comma(int64(result.Rows)),
					english.PluralWord(result.Rows, "event", ""),
					humanize.Comma(int64(rowsPerSecond)),
					txtDuration,
				)
			} else {
				textMessage = fmt.Sprintf("[%d] %s - %s - %s processed %s %s%s", wid, source.Prefix, result.Instance, result.Session,
					humanize.Comma(int64(result.Rows)),
					english.PluralWord(result.Rows, "event", ""),
					txtDuration,
				)
			}
			if result.Rows > 0 {
				_ = applog.Info(textMessage)
			}
		}

		// TODO - Write the message here
		log.Println(textMessage)
	}

	// Process Agent Jobs
	if source.AgentJobs == config.JobsAll || source.AgentJobs == config.JobsFailed {
		start := time.Now()
		result, err := processAgentJobs(wid, source)
		runtime := time.Since(start)
		totalSeconds := runtime.Seconds()
		// TODO - based on the error, generate a message

		var rowsPerSecond int64
		if totalSeconds > 0.0 {
			rowsPerSecond = int64(float64(result.Rows) / totalSeconds)
		}
		sourceResult.Rows += result.Rows

		var textMessage string

		if err != nil {
			textMessage = fmt.Sprintf("[%d] *** ERROR: Prefix: %s; FQDN: %s; (%s) %s\r\n", wid, source.Prefix, source.FQDN, "Agent Jobs", err.Error())
			cleanRun = false
			_ = applog.Error(textMessage)
		} else {
			if rowsPerSecond > 0 && totalSeconds > 1 {
				textMessage = fmt.Sprintf("[%d] %s - %s - %s processed %s %s - %s per second",
					wid, source.Prefix, result.Instance, result.Session,
					humanize.Comma(int64(result.Rows)),
					english.PluralWord(result.Rows, "event", ""),
					humanize.Comma(int64(rowsPerSecond)))
			} else {
				textMessage = fmt.Sprintf("[%d] %s - %s - %s processed %s %s", wid, source.Prefix, result.Instance, result.Session,
					humanize.Comma(int64(result.Rows)),
					english.PluralWord(result.Rows, "event", ""))
			}
			if result.Rows > 0 {
				_ = applog.Info(textMessage)
			}
		}

		// TODO - Write the message here
		log.Println(textMessage)
	}

	if !cleanRun {
		err = errors.New("errors occured - see previous")
	}
	return sourceResult, err
}
