package main

import (
	"fmt"
	"net"

	"github.com/billgraziano/xelogstash/config"
	"github.com/billgraziano/xelogstash/log"
	"github.com/billgraziano/xelogstash/logstash"
	"github.com/billgraziano/xelogstash/status"
	"github.com/billgraziano/xelogstash/summary"
	"github.com/billgraziano/xelogstash/xe"
	"github.com/pkg/errors"
)

func processSession(
	wid int,
	info xe.SQLInfo,
	source config.Source,
	sessionid int) (result Result, err error) {

	result.Session = source.Sessions[sessionid]
	result.Source = source
	result.Instance = source.FQDN // this will be reset later

	var objectName, eventData, fileName string
	var fileOffset int64

	var lastFileName string
	var lastFileOffset int64
	var xestatus string

	result.Instance = info.Server

	err = status.SwitchV2(wid, source.Prefix, info.Domain, info.Server, status.ClassXE, result.Session)
	if err != nil {
		return result, errors.Wrap(err, "status.switchv2")
	}

	// do the dupe check based on the actual instance since that's what is stored
	err = status.CheckDupe(info.Domain, result.Instance, status.ClassXE, result.Session)
	if err != nil {
		return result, errors.Wrap(err, "dupe.check")
	}

	if err = xe.ValidateSession(info.DB, result.Session); err != nil {
		return result, errors.Wrap(err, "validatesession")
	}

	session, err := xe.GetSession(info.DB, result.Session)
	if err != nil {
		return result, errors.Wrap(err, "xe.getsession")
	}

	sf, err := status.NewFile(info.Domain, result.Instance, status.ClassXE, result.Session)
	if err != nil {
		return result, errors.Wrap(err, "status.newfile")
	}
	lastFileName, lastFileOffset, xestatus, err = sf.GetOffset()
	if err != nil {
		return result, errors.Wrap(err, "status.getoffset")
	}

	if xestatus == status.StateReset {
		log.Error(fmt.Sprintf("[%d] *** ERROR ***", wid))
		log.Error(fmt.Sprintf("[%d] *** Missing events in previous run from: [%s-%s-%s] starting at [%s-%d]", wid, info.Domain, result.Instance, result.Session, lastFileName, lastFileOffset))
		log.Error(fmt.Sprintf("[%d] *** Attempting to read past this offset.  Events are probably missed.", wid))
		// returnErr = errors.New("Recovering from missing events")
		// TODO Log to logstash with error
	}

	var query string
	var ls *logstash.Logstash
	if appConfig.Logstash != "" {
		ls, err = logstash.NewHost(appConfig.Logstash, 180)
		if err != nil {
			return result, errors.Wrap(err, "logstash-new")
		}
	}

	if (lastFileName == "" && lastFileOffset == 0) || xestatus == status.StateReset {
		query = fmt.Sprintf("SELECT object_name, event_data, file_name, file_offset FROM sys.fn_xe_file_target_read_file('%s', NULL, NULL, NULL);", session.WildCard)
	} else {
		query = fmt.Sprintf("SELECT object_name, event_data, file_name, file_offset FROM sys.fn_xe_file_target_read_file('%s', NULL, '%s', %d);", session.WildCard, lastFileName, lastFileOffset)
	}
	rows, err := info.DB.Query(query)
	if err != nil {
		// if we have a file name, save it with a reset status
		if len(lastFileName) > 0 {
			saveErr := sf.Done(lastFileName, lastFileOffset, status.StateReset)
			if saveErr != nil {
				log.Error(fmt.Sprintf("[%d] Error saving the status file: %v", wid, saveErr))

			}
		}
		log.Error(fmt.Sprintf("[%d] *** Read XE file target error occurred.", wid))
		log.Error(fmt.Sprintf("[%d] *** The next run will attempt to read past this error.  Events may be skipped.", wid))

		// TODO log this error
		return result, errors.Wrap(err, "query")
	}

	var netconn *net.TCPConn
	first := true
	gotRows := false

	for rows.Next() {
		if first && ls != nil {
			netconn, err = ls.Connect()
			if err != nil {
				return result, errors.Wrap(err, "logstash-connect")
			}
			defer safeClose(netconn, &err)
		}

		//start := time.Now()

		err = rows.Scan(&objectName, &eventData, &fileName, &fileOffset)
		if err != nil {
			return result, errors.Wrap(err, "scan")
		}

		// We had an issue last time, we are starting from scratch to find a good offset
		// We read until we get past a point we already had
		if xestatus == status.StateReset {
			if fileName < lastFileName { // we are reading an earlier file
				continue
			}
			if fileName == lastFileName && fileOffset <= lastFileOffset { // we already have this
				continue
			}
		}

		if first {
			lastFileName = fileName
			lastFileOffset = fileOffset
		}

		gotRows = true

		// Did we just finish a file offset
		if fileName != lastFileName || fileOffset != lastFileOffset {
			//if !source.Test {
			err = sf.Save(lastFileName, lastFileOffset, status.StateSuccess)
			if err != nil {
				return result, errors.Wrap(err, "status.Save")
			}
			//}

			// do we have as many rows as we need?
			if source.Rows > 0 && result.Rows >= source.Rows {
				break
			}
		}

		lastFileName = fileName
		lastFileOffset = fileOffset

		first = false

		var event xe.Event
		event, err = xe.Parse(&info, eventData)
		if err != nil {
			return result, errors.Wrap(err, "xe.Parse")
		}

		// is this an event we are skipping?
		// TODO Lower this into the Parse function
		eventName := event.Name()
		if containsString(source.ExcludedEvents, eventName) {
			continue
		}

		// add default columns
		event.Set("xe_session_name", result.Session)
		event.Set("xe_file_name", fileName)
		event.Set("xe_file_offset", fileOffset)
		//event.SetAppSource()

		lr := logstash.NewRecord()
		// if payload field is empty
		if source.PayloadField == "" {
			for k, v := range event {
				lr[k] = v
			}
		} else {
			//fmt.Println(source.PayloadField)
			lr[source.PayloadField] = event
			lr[source.TimestampField] = event["timestamp"]
		}
		//fmt.Println("tsfield:", source.TimestampField)
		if source.TimestampField != "timestamp" && source.PayloadField == "" {
			lr[source.TimestampField] = event["timestamp"]
			delete(lr, "timestamp")
		}

		var rs string
		rs, err = lr.ToJSON()
		if err != nil {
			return result, errors.Wrap(err, "record.tojson")
		}

		// process the adds and such
		rs, err = logstash.ProcessMods(rs, source.Adds, source.Copies, source.Moves)
		if err != nil {
			return result, errors.Wrap(err, "logstash.processmods")
		}

		if ls != nil {
			err = ls.Writeln(rs)
			if err != nil {
				log.Error("")
				log.Error(fmt.Sprintf("%s", rs))
				log.Error("")
				return result, errors.Wrap(err, "logstash-writeln")
			}
		}

		//fmt.Printf("\r\n\r\n%s\r\n\r\n", rs)

		result.Rows++
		totalCount.Add(1)
		eventCount.Add(eventName, 1)
		if appConfig.Summary {
			//summary.Add(eventName, &eventString)
			summary.Add(eventName, &rs)
		}
	}

	err = rows.Err()
	if err != nil {
		return result, errors.Wrap(err, "rows.Err")
	}

	if gotRows /* && !source.Test */ {
		err = sf.Save(lastFileName, lastFileOffset, status.StateSuccess)
		if err != nil {
			return result, errors.Wrap(err, "status.Save")
		}

		err = sf.Done(lastFileName, lastFileOffset, status.StateSuccess)
		if err != nil {
			return result, errors.Wrap(err, "status.done")
		}
	}

	return result, nil
}
