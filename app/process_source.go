package app

import (
	"context"
	"fmt"
	"time"

	_ "github.com/alexbrainman/odbc"
	"github.com/billgraziano/xelogstash/config"
	"github.com/billgraziano/xelogstash/status"
	"github.com/billgraziano/xelogstash/xe"
	humanize "github.com/dustin/go-humanize"
	"github.com/dustin/go-humanize/english"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// ProcessSource handles the sessions and jobs for an instance
func (p *Program) ProcessSource(ctx context.Context, wid int, source config.Source) (sourceResult Result, err error) {

	logmsg := fmt.Sprintf("[%d] Source: %s;  Sessions: %d", wid, source.FQDN, len(source.Sessions))
	if source.Exclude17830 {
		logmsg += ";  Excluding 17830 errors"
	}

	if !source.StartAt.IsZero() {
		logmsg += fmt.Sprintf(";  Events after: %v", source.StartAt.Format(time.RFC3339))
	}

	if source.StopAt != config.DefaultStopAt {
		logmsg += fmt.Sprintf("; Events before: %v", source.StopAt.Format(time.RFC3339))
	}
	log.Debug(logmsg)

	var textMessage string
	info, err := xe.GetSQLInfo(source.FQDN)
	if err != nil {
		textMessage = fmt.Sprintf("[%d] %s - fqdn: %s err: %v", wid, info.Domain, source.FQDN, err)
		log.Error(textMessage)
		return sourceResult, errors.Wrap(err, "xe.getsqlinfo")
	}

	defer safeClose(info.DB, &err)

	cleanRun := true
	for i := range source.Sessions {
		if ctx.Err() != nil {
			break
		}
		log.Trace(fmt.Sprintf("[%d] Starting session: %s on  %s", wid, source.Sessions[i], source.FQDN))
		start := time.Now()

		var result Result
		result, err = p.processSession(ctx, wid, info, source, i)
		runtime := time.Since(start)
		totalSeconds := runtime.Seconds()

		var rowsPerSecond int64
		if totalSeconds > 0.0 {
			rowsPerSecond = int64(float64(result.Rows) / totalSeconds)
		}
		sourceResult.Instance = result.Instance
		sourceResult.Rows += result.Rows

		//txtDuration := fmt.Sprintf(" (%s)", format.RoundDuration(runtime, time.Second))
		// if runtime.Seconds() < 10 {
		// 	txtDuration = ""
		// }

		if errors.Cause(err) == xe.ErrNotFound {
			// TODO: if strict, then warning:
			// textMessage = fmt.Sprintf("[%d] %s - %s - %s is not available.  Skipped.", wid, source.Prefix, result.Instance, result.Session)
			// else
			continue
		} else if errors.Cause(err) == xe.ErrNoFileTarget && source.Sessions[i] == "system_health" {
			// no file target on system_health is a warning (#36)
			textMessage = fmt.Sprintf("[%d] Domain: %s - FQDN: %s - %s - %s : %s\r\n", wid, info.Domain, source.FQDN, status.ClassXE, source.Sessions[i], err.Error())
			log.Info(textMessage)
		} else if err != nil {
			textMessage = fmt.Sprintf("[%d] *** ERROR: Domain: %s - FQDN: %s - %s - %s : %s\r\n", wid, info.Domain, source.FQDN, status.ClassXE, source.Sessions[i], err.Error())
			cleanRun = false
			log.Error(textMessage)
		} else {

			// if rowsPerSecond > 0 && totalSeconds > 1 {
			// 	textMessage = fmt.Sprintf("[%d] %s - %s - %s - %s %s - %s per second%s",
			// 		wid,
			// 		info.Domain,
			// 		result.Instance,
			// 		result.Session,
			// 		humanize.Comma(int64(result.Rows)),
			// 		english.PluralWord(result.Rows, "event", ""),
			// 		humanize.Comma(int64(rowsPerSecond)),
			// 		txtDuration,
			// 	)
			// } else {
			// 	textMessage = fmt.Sprintf("[%d] %s - %s - %s - %s %s%s",
			// 		wid,
			// 		info.Domain,
			// 		result.Instance,
			// 		result.Session,
			// 		humanize.Comma(int64(result.Rows)),
			// 		english.PluralWord(result.Rows, "event", ""),
			// 		txtDuration,
			// 	)
			// }
			// [2] WORKGROUP - D40\SQL2017 - system_health - 100 events - 50 per second
			// server: D40\SQL2017 (WORKGROUP) session: system_health events: 100  per_second: 50
			if rowsPerSecond > 0 && totalSeconds > 1 {
				textMessage = fmt.Sprintf("%s (%s) session: %s events: %s per_second: %s",
					result.Instance,
					info.Domain,
					result.Session,
					humanize.Comma(int64(result.Rows)),
					humanize.Comma(int64(rowsPerSecond)),
				)
			} else {
				textMessage = fmt.Sprintf("%s (%s) session: %s events: %s ",
					result.Instance,
					info.Domain,
					result.Session,
					humanize.Comma(int64(result.Rows)),
				)
			}
			if p.Verbose {
				log.Info(textMessage)
			} else {
				log.Debug(textMessage)
			}
		}
	}

	if ctx.Err() != nil {
		return sourceResult, nil
	}

	// Process Agent Jobs
	if source.AgentJobs == config.JobsAll || source.AgentJobs == config.JobsFailed {
		start := time.Now()

		var result Result
		result, err = p.processAgentJobs(ctx, wid, source)
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
			textMessage = fmt.Sprintf("[%d] *** ERROR: Domain: %s; FQDN: %s; (%s) %s\r\n", wid, info.Domain, source.FQDN, "Agent Jobs", err.Error())
			cleanRun = false
			log.Error(textMessage)
		} else {
			if rowsPerSecond > 0 && totalSeconds > 1 {
				textMessage = fmt.Sprintf("[%d] %s - %s - %s - %s %s - %s per second",
					wid, info.Domain, result.Instance, result.Session,
					humanize.Comma(int64(result.Rows)),
					english.PluralWord(result.Rows, "event", ""),
					humanize.Comma(int64(rowsPerSecond)))
			} else {
				textMessage = fmt.Sprintf("[%d] %s - %s - %s - %s %s", wid, info.Domain, result.Instance, result.Session,
					humanize.Comma(int64(result.Rows)),
					english.PluralWord(result.Rows, "event", ""))
			}
			log.Debug(textMessage)
		}
	}

	if !cleanRun {
		err = errors.New("errors occurred - see previous")
	}
	return sourceResult, err
}
