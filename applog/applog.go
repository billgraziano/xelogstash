package applog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/billgraziano/go-elasticsearch"
	"github.com/billgraziano/xelogstash/config"
	"github.com/billgraziano/xelogstash/eshelper"
	"github.com/billgraziano/xelogstash/log"
	"github.com/billgraziano/xelogstash/logstash"
	"github.com/billgraziano/xelogstash/seq"
	"github.com/pkg/errors"
)

// appconfig holds the AppLog appconfig
var cfg config.Config
var mux sync.Mutex
var ls *logstash.Logstash

var elasticBuffer bytes.Buffer
var esclient *elasticsearch.Client

var logs []string

// Initialize configures the app logging
func Initialize(c config.Config) (err error) {
	cfg = c

	mux.Lock()
	defer mux.Unlock()

	if cfg.AppLog.Logstash != "" {
		ls, err = logstash.NewHost(cfg.AppLog.Logstash, 180)
		if err != nil {
			return errors.Wrap(err, "logstash.newhost")
		}

		_, err = ls.Connect()
		if err != nil {
			return errors.Wrap(err, "logstash-connect")
		}
	}

	// Setup the Elastic client
	if len(cfg.Elastic.Addresses) > 0 && cfg.Elastic.Username != "" && cfg.Elastic.Password != "" {
		esclient, err = eshelper.NewClient(cfg.Elastic.Addresses, cfg.Elastic.ProxyServer, cfg.Elastic.Username, cfg.Elastic.Password)
		if err != nil {
			return errors.Wrap(err, "eshelper.newclient")
		}
	}

	return nil
}

// Close any open TCP connections
func Close() error {
	if ls != nil {
		if ls.Connection != nil {
			err := ls.Close()
			if err != nil {
				return errors.Wrap(err, "ls.close")
			}
		}
	}
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
	if cfg.AppLog.PayloadField == "" {
		for k, v := range src {
			lr[k] = v
		}
	} else {
		lr[cfg.AppLog.PayloadField] = src
	}

	if cfg.AppLog.TimestampField == "" {
		lr["timestamp"] = now
	} else {
		lr[cfg.AppLog.TimestampField] = now
	}

	rs, err := lr.ToJSON()
	if err != nil {
		return errors.Wrap(err, "record.tojson")
	}

	// process the adds and such
	rs, err = logstash.ProcessMods(rs, cfg.AppLog.Adds, cfg.AppLog.Copies, cfg.AppLog.Moves)
	if err != nil {
		return errors.Wrap(err, "logstash.processmods")
	}

	mux.Lock()
	defer mux.Unlock()

	logs = append(logs, rs)

	if cfg.AppLog.Logstash != "" {

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
	}

	// Write one entry to the buffer
	if esclient != nil && cfg.Elastic.AppLogIndex != "" {
		meta := []byte(fmt.Sprintf(`{ "index" : { "_index" : "%s" } }%s`, cfg.Elastic.AppLogIndex, "\n"))
		espayload := []byte(rs + "\n")
		elasticBuffer.Grow(len(meta) + len(espayload))
		elasticBuffer.Write(meta)
		elasticBuffer.Write(espayload)

		err = eshelper.WriteElasticBuffer(esclient, &elasticBuffer)
		if err != nil {
			return errors.Wrap(err, "eshelper.writeelasticbuffer")
		}
	}

	return nil
}

// GetLogFile returns a log file pointer for writing
func GetLogFile(name string) (*os.File, error) {
	// Get EXE directory
	executable, err := os.Executable()
	if err != nil {
		return nil, errors.Wrap(err, "os.executable")
	}
	exeDir := filepath.Dir(executable)
	fileName := filepath.Join(exeDir, name)

	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return nil, errors.Wrap(err, "os.openfile")
	}
	return file, nil
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
