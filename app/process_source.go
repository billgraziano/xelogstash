package app

import (
	"context"
	"fmt"
	"time"

	"github.com/billgraziano/mssqlh"
	"github.com/billgraziano/xelogstash/config"
	"github.com/billgraziano/xelogstash/status"
	"github.com/billgraziano/xelogstash/xe"
	humanize "github.com/dustin/go-humanize"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// ProcessSource handles the sessions and jobs for an instance
func (p *Program) ProcessSource(ctx context.Context, wid int, source config.Source) (sourceResult Result, err error) {

	contextLogger := log.WithFields(log.Fields{
		"source": source.FQDN,
	})

	contextLogger.Trace("entering ProcessSource")

	logmsg := fmt.Sprintf("source: %s;  sessions: %d", source.FQDN, len(source.Sessions))
	if source.Exclude17830 {
		logmsg += ";  exclude 17830 errors"
	}

	if !source.StartAt.IsZero() {
		logmsg += fmt.Sprintf(";  events after: %v", source.StartAt.Format(time.RFC3339))
	}

	if source.StopAt != config.DefaultStopAt {
		logmsg += fmt.Sprintf("; events before: %v", source.StopAt.Format(time.RFC3339))
	}
	contextLogger.Trace(logmsg)

	var textMessage string
	if source.User != "" {
		contextLogger.Debugf("user: %s", source.User)
	}

	cxn := mssqlh.NewConnection(source.FQDN, source.User, source.Password, "master", "sqlxewriter.exe")
	if source.Driver != "" {
		cxn.Driver = source.Driver
	}
	if source.ODBCDriver != "" {
		cxn.ODBCDriver = source.ODBCDriver
	}
	info, err := xe.GetSQLInfo(cxn.Driver, cxn.String())
	if err != nil {
		textMessage = fmt.Sprintf("source: %s err: %v", source.FQDN, err)
		contextLogger.Error(textMessage)
		return sourceResult, errors.Wrap(err, "xe.getsqlinfo")
	}

	defer safeClose(info.DB, &err)
	contextLogger = contextLogger.WithFields(log.Fields{
		"instance": info.Server,
	})

	contextLogger.Tracef("%s: sys.messages for login_failed: %d\n", info.Server, len(info.LoginErrors))

	cleanRun := true
	for i := range source.Sessions {
		sessionLogger := contextLogger.WithFields(log.Fields{
			"session": source.Sessions[i],
		})
		if ctx.Err() != nil {
			break
		}
		sessionLogger.Trace(fmt.Sprintf("%s: starting: session: %s", source.FQDN, source.Sessions[i]))
		start := time.Now()

		var result Result
		result, err = p.processSession(ctx, wid, info, source, i)
		runtime := time.Since(start)
		totalSeconds := runtime.Seconds()
		totalMilliseconds := runtime.Milliseconds()

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
			textMessage = fmt.Sprintf("source: %s (%s) - %s - %s : %s", source.FQDN, info.Domain, status.ClassXE, source.Sessions[i], err.Error())
			sessionLogger.Info(textMessage)
		} else if err != nil {
			textMessage = fmt.Sprintf("source: %s (%s) - %s - %s : %s", source.FQDN, info.Domain, status.ClassXE, source.Sessions[i], err.Error())
			cleanRun = false
			sessionLogger.Error(textMessage)
		} else {
			// server: D40\SQL2017 (WORKGROUP) session: system_health events: 100  per_second: 50
			if rowsPerSecond > 0 && totalSeconds > 1 {
				textMessage = fmt.Sprintf("%s (%s) session: %s; events: %s; per_second: %s",
					result.Instance,
					info.Domain,
					result.Session,
					humanize.Comma(int64(result.Rows)),
					humanize.Comma(int64(rowsPerSecond)),
				)
			} else {
				textMessage = fmt.Sprintf("%s (%s) session: %s; events: %s",
					result.Instance,
					info.Domain,
					result.Session,
					humanize.Comma(int64(result.Rows)),
				)
			}
			if p.Verbose {
				sessionLogger.WithFields(log.Fields{
					"events":         result.Rows,
					"duration_ms":    totalMilliseconds,
					"events_per_sec": rowsPerSecond,
				}).Info(textMessage)
			} else {
				sessionLogger.WithFields(log.Fields{
					"events":         result.Rows,
					"duration_ms":    totalMilliseconds,
					"events_per_sec": rowsPerSecond,
				}).Debug(textMessage)
			}
		}
	}

	if ctx.Err() != nil {
		return sourceResult, nil
	}

	// Process Agent Jobs
	if source.AgentJobs == config.JobsAll || source.AgentJobs == config.JobsFailed {
		start := time.Now()

		sessionLogger := contextLogger.WithFields(log.Fields{
			"session": "agent_jobs",
		})

		var result Result
		result, err = p.processAgentJobs(ctx, wid, source)
		runtime := time.Since(start)
		totalSeconds := runtime.Seconds()
		totalMilliseconds := runtime.Milliseconds()
		// TODO - based on the error, generate a message

		var rowsPerSecond int64
		if totalSeconds > 0.0 {
			rowsPerSecond = int64(float64(result.Rows) / totalSeconds)
		}
		sourceResult.Rows += result.Rows

		var textMessage string

		if err != nil {
			textMessage = fmt.Sprintf("source: %s (%s); (%s) %s\r\n", source.FQDN, info.Domain, "agent_jobs", err.Error())
			cleanRun = false
			sessionLogger.Error(textMessage)
		} else {
			if rowsPerSecond > 0 && totalSeconds > 1 {
				textMessage = fmt.Sprintf("%s (%s) session: agent_jobs; events: %s; per_second: %s",
					result.Instance,
					info.Domain,
					humanize.Comma(int64(result.Rows)),
					humanize.Comma(int64(rowsPerSecond)))
			} else {
				textMessage = fmt.Sprintf("%s (%s) session: agent_jobs; events: %s",
					result.Instance,
					info.Domain,
					humanize.Comma(int64(result.Rows)))
			}
			if p.Verbose {
				sessionLogger.WithFields(log.Fields{
					"events":         result.Rows,
					"duration_ms":    totalMilliseconds,
					"events_per_sec": rowsPerSecond,
				}).Info(textMessage)
			} else {
				sessionLogger.WithFields(log.Fields{
					"events":         result.Rows,
					"duration_ms":    totalMilliseconds,
					"events_per_sec": rowsPerSecond,
				}).Debug(textMessage)
			}
		}
	}

	if !cleanRun {
		err = errors.New("errors occurred - see previous")
	}
	return sourceResult, err
}
