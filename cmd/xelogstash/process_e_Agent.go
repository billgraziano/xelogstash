package main

import (
	"database/sql"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/billgraziano/mssqlodbc"
	"github.com/billgraziano/xelogstash/config"
	"github.com/billgraziano/xelogstash/log"
	"github.com/billgraziano/xelogstash/logstash"
	"github.com/billgraziano/xelogstash/status"
	"github.com/billgraziano/xelogstash/summary"
	"github.com/pkg/errors"
)

type jobResult struct {
	Name          string `json:"name"`
	InstanceID    int    `json:"instance_id"`
	JobID         string `json:"job_id"`
	StepID        int    `json:"step_id"`
	JobName       string `json:"job_name"`
	Message       string `json:"message"`
	RunStatus     int    `json:"run_status"`
	RunStatusText string
	RunDuration   int       `json:"run_duration"`
	Timestamp     time.Time `json:"timestamp"`
	FQDN          string    `json:"mssql_fqdn"`
	Computer      string    `json:"mssql_computer"`
	Server        string    `json:"mssql_server_name"`
	Version       string    `json:"mssql_version"`
	Domain        string    `json:"mssql_domain"`
}

func processAgentJobs(wid int, source config.Source) (result Result, err error) {

	result.Session = "agent_jobs"
	result.Instance = source.FQDN // this will be reset later
	result.Source = source
	dummyFileName := "_dummy_"

	cxn := mssqlodbc.Connection{
		Server:  source.FQDN,
		AppName: "xelogstash.exe",
		Trusted: true,
	}

	connectionString, err := cxn.ConnectionString()
	if err != nil {
		return result, errors.Wrap(err, "mssqlodbc.connectionstring")
	}
	db, err := sql.Open("odbc", connectionString)
	if err != nil {
		return result, errors.Wrap(err, "db.open")
	}

	err = db.Ping()
	if err != nil {
		return result, errors.Wrap(err, "db.ping")
	}
	defer safeClose(db, &err)

	info, err := GetInstance(db, source.FQDN)
	if err != nil {
		return result, errors.Wrap(err, "getinstance")
	}
	result.Instance = info.Server

	// do the dupe check based on the actual instance since that's what is stored
	err = status.CheckDupe(source.Prefix, result.Instance, status.ClassAgentJobs, result.Session)
	if err != nil {
		return result, errors.Wrap(err, "dupe.check")
	}

	//appStart := time.Now()

	sf, err := status.NewFile(source.Prefix, result.Instance, status.ClassAgentJobs, result.Session)
	if err != nil {
		return result, errors.Wrap(err, "status.newfile")
	}
	_, lastInstanceID, _, err := sf.GetOffset()
	if err != nil {
		return result, errors.Wrap(err, "status.getoffset")
	}

	var ls *logstash.Logstash
	if appConfig.Logstash != "" {
		ls, err = logstash.NewHost(appConfig.Logstash, 180)
		if err != nil {
			return result, errors.Wrap(err, "logstash-new")
		}
	}

	var query string
	top := 100000
	if source.Rows > 0 {
		top = source.Rows
	}
	topClause := fmt.Sprintf(" TOP (%d) ", top)
	query = "SELECT " + topClause
	query += `
				CASE 
				WHEN H.step_id = 0 THEN 'agent_job' 
				ELSE 'agent_job_step'
			END AS [name]
			,H.instance_id
			,H.job_id
			,H.step_id
			,J.[name] AS [job_name]
			,H.[message] as [msg]
			,H.[run_status]
			,H.[run_duration]
			,convert( datetime,
				SUBSTRING(CAST(run_date AS VARCHAR(8)),1,4) + '-' + 
				SUBSTRING(CAST(run_date AS VARCHAR(8)),5,2) + '-'+ 
				SUBSTRING(CAST(run_date AS VARCHAR(8)),7,2) + ' ' +
				convert(varchar, run_time/10000)+':'+
				convert(varchar, run_time%10000/100)+':'+
				convert(varchar, run_time%100)+'.000' ) AS [Timestamp]


		FROM	[msdb].[dbo].[sysjobhistory] H
		JOIN	[msdb].[dbo].[sysjobs] J on J.[job_id] = H.[job_id]
		WHERE	1=1
		-- AND  H.[run_status] NOT IN (2, 4) -- Don't want retry or in progress
		AND		H.[instance_id] > ?
		ORDER BY H.[instance_id] ASC;
	`

	rows, err := db.Query(query, lastInstanceID)
	if err != nil {
		return result, errors.Wrap(err, "db.open")
	}

	var netconn *net.TCPConn
	first := true
	//gotRows := false
	//var rowCount int
	var instanceID int
	var gotRows bool

	for rows.Next() {
		if first && ls != nil {
			netconn, err = ls.Connect()
			if err != nil {
				return result, errors.Wrap(err, "logstash-connect")
			}
			defer safeClose(netconn, &err)
		}

		var j jobResult
		err = rows.Scan(&j.Name, &j.InstanceID, &j.JobID, &j.StepID, &j.JobName, &j.Message, &j.RunStatus, &j.RunDuration, &j.Timestamp)
		if err != nil {
			return result, errors.Wrap(err, "rows.scan")
		}
		instanceID = j.InstanceID

		// TODO do the copies, adds, renames

		// // write to log stash
		// b, err := json.Marshal(j)
		// if err != nil {
		// 	return result, errors.Wrap(err, "json.marshal")
		// }
		// recordString := string(b)

		base := logstash.NewRecord()
		base.Set("name", j.Name)
		base.Set("instance_id", j.InstanceID)
		base.Set("job_id", j.JobID)
		base.Set("step_id", j.StepID)
		base.Set("job_name", j.JobName)
		base.Set("message", j.Message)
		base.Set("run_status", j.RunStatus)
		switch j.RunStatus {
		case 0:
			base.Set("run_status_text", "failed")
			base.Set("xe_severity_value", logstash.Error)
			base.Set("xe_severity_keyword", logstash.Error.String())
		case 1:
			base.Set("run_status_text", "succeeded")
			base.Set("xe_severity_value", logstash.Info)
			base.Set("xe_severity_keyword", logstash.Info.String())
		case 2:
			base.Set("run_status_text", "retry")
			base.Set("xe_severity_value", logstash.Warning)
			base.Set("xe_severity_keyword", logstash.Warning.String())
		case 3:
			base.Set("run_status_text", "cancelled")
			base.Set("xe_severity_value", logstash.Warning)
			base.Set("xe_severity_keyword", logstash.Warning.String())
		default:
			base.Set("run_status_text", "unknown")
			base.Set("xe_severity_value", logstash.Warning)
			base.Set("xe_severity_keyword", logstash.Warning.String())
		}
		base.Set("run_duration", j.RunDuration)
		base.Set("timestamp", j.Timestamp)

		base.Set("mssql_domain", info.Domain)
		base.Set("mssql_computer", info.Computer)
		base.Set("mssql_server_name", info.Server)
		base.Set("mssql_version", info.Version)

		base.SetIfEmpty("server_instance_name", info.Server)

		// set the description
		var description string
		switch j.Name {
		case "agent_job":
			description = j.Message
		case "agent_job_step":
			description = j.Message
		}
		if len(description) > 0 {
			base.Set("xe_description", description)
		}

		//base.ToLower()

		lr := logstash.NewRecord()
		// if payload field is empty
		if source.PayloadField == "" {
			for k, v := range base {
				lr[k] = v
			}
		} else {
			//fmt.Println(source.PayloadField)
			lr[source.PayloadField] = base
			lr[source.TimestampField] = base["timestamp"]
		}
		// if we're in the root
		if source.TimestampField != "timestamp" && source.PayloadField == "" {
			lr[source.TimestampField] = base["timestamp"]
			delete(lr, "timestamp")
		}

		rs, err := lr.ToJSON()
		if err != nil {
			return result, errors.Wrap(err, "record.tojson")
		}

		// process the adds and such
		rs, err = logstash.ProcessMods(rs, source.Adds, source.Copies, source.Moves)
		if err != nil {
			return result, errors.Wrap(err, "logstash.processmods")
		}

		// TODO if test, write {}
		if ls != nil {
			err = ls.Writeln(rs)
			if err != nil {
				log.Printf("\r\n")
				log.Printf("%s\r\n", rs)
				log.Printf("\r\n")
				return result, errors.Wrap(err, "logstash-writeln")
			}
		}

		if appConfig.Summary {
			summary.Add(j.Name, &rs)
		}

		// write the status field
		err = sf.Save(dummyFileName, int64(j.InstanceID), status.StatusSuccess)
		if err != nil {
			return result, errors.Wrap(err, "status.Save")
		}

		first = false
		gotRows = true
		//rowCount++
		result.Rows++
	}

	if gotRows {
		err = sf.Done(dummyFileName, int64(instanceID), status.StatusSuccess)
		if err != nil {
			return result, errors.Wrap(err, "status.done")
		}
	}

	return result, nil
}

func jobToRecord(j jobResult) (r logstash.Record, err error) {
	//r.Event = j
	//r.Timestamp = j.Timestamp

	return r, err
}

func parseAgentTime(d, t int) (time.Time, error) {

	dt := strconv.Itoa(d)
	tm := "000000" + strconv.Itoa(t)
	tm = tm[len(tm)-6:]
	fmt.Println(dt, tm, dt+tm)
	v, err := time.Parse("20060102150405", dt+tm)

	if err != nil {
		return v, err
	}
	return v, nil
}
