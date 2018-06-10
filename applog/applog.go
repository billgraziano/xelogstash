package applog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/billgraziano/xelogstash/config"
	"github.com/billgraziano/xelogstash/log"
	"github.com/billgraziano/xelogstash/logstash"
	"github.com/billgraziano/xelogstash/seq"
	"github.com/pkg/errors"
)

// appconfig holds the AppLog appconfig
var appconfig config.AppLog
var mux sync.Mutex
var ls *logstash.Logstash

var logs []string

// Initialize configures the app logging
func Initialize(c config.AppLog) (err error) {
	appconfig = c

	if appconfig.Logstash == "" {
		return nil
	}

	ls, err = logstash.NewHost(appconfig.Logstash, 180)
	if err != nil {
		return errors.Wrap(err, "logstash.newhost")
	}

	//var netconn *net.TCPConn
	_, err = ls.Connect()
	if err != nil {
		return errors.Wrap(err, "logstash-connect")
	}

	//fmt.Println(ls.Connection, ls.Timeout, ls.Host)

	return nil
}

// Error logs an error
func Error(msg string) (err error) {
	r := logstash.NewRecord()
	r["message"] = msg
	r["severity"] = logstash.Error.String()
	err = Log(r)
	if err != nil {
		log.Error("applog to logstash failed:", err)
	}
	return nil
}

// Warn logs an error
func Warn(msg string) (err error) {
	r := logstash.NewRecord()
	r["message"] = msg
	r["severity"] = logstash.Warning.String()
	err = Log(r)
	if err != nil {
		log.Error("applog to logstash failed:", err)
	}
	return nil
}

// Info writes a single string to logstash
func Info(msg string) error {
	var err error
	r := logstash.NewRecord()
	r["message"] = msg
	err = Log(r)
	if err != nil {
		log.Error("applog to logstash failed:", err)
	}
	return nil
}

// Log writes a log to logstash
func Log(src logstash.Record) error {
	mux.Lock()
	defer mux.Unlock()

	//fmt.Println(ls)

	//defer netconn.Close()

	////////////////////////////////
	// Do all my stuff here
	////////////////////////////////

	now := time.Now().UTC()
	lr := logstash.NewRecord()

	// Check for severity
	_, ok := src["severity"]
	if !ok {
		src["severity"] = logstash.Info.String()
	}

	src["sequence_value"] = seq.Get()

	// if payload field is empty
	if appconfig.PayloadField == "" {
		for k, v := range src {
			lr[k] = v
		}
	} else {
		lr[appconfig.PayloadField] = src
	}

	if appconfig.TimestampField == "" {
		lr["timestamp"] = now
	} else {
		lr[appconfig.TimestampField] = now
	}

	rs, err := lr.ToJSON()
	if err != nil {
		return errors.Wrap(err, "record.tojson")
	}

	// process the adds and such
	rs, err = logstash.ProcessMods(rs, appconfig.Adds, appconfig.Copies, appconfig.Moves)
	if err != nil {
		return errors.Wrap(err, "logstash.processmods")
	}

	logs = append(logs, rs)

	if appconfig.Logstash == "" {
		return nil
	}

	if ls.Connection == nil {
		log.Debug("ls.connection is nil")
	}

	//fmt.Println(rs)
	err = ls.Writeln(rs)
	if err != nil {
		log.Error("")
		log.Error(fmt.Sprintf("%s", rs))
		log.Error("")
		return errors.Wrap(err, "logstash-writeln")
	}

	return nil
}

// PrintSamples prints out the JSON log messages
func PrintSamples() error {
	// Get EXE directory
	executable, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "os.executable")
	}
	exeDir := filepath.Dir(executable)
	fileName := filepath.Join(exeDir, "samples.applog.json")

	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return errors.Wrap(err, "os.openfile")
	}
	defer file.Close()

	for _, v := range logs {
		//json, err := json.Unmarshal(v.Sample)
		// z := bytes.NewBufferString(v.Sample)
		var out bytes.Buffer
		err = json.Indent(&out, []byte(v), "", "  ")
		if err != nil {
			return errors.Wrap(err, "json.indent")
		}

		if _, err = file.Write([]byte(fmt.Sprintf("%s\r\n\r\n", out.String()))); err != nil {
			return errors.Wrap(err, "file.write")
		}
	}
	err = file.Sync()
	if err != nil {
		return errors.Wrap(err, "file.sync")
	}

	return nil
}
