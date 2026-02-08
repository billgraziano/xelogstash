package main

import (
	"flag"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"runtime"
	"time"

	"github.com/billgraziano/mssqlh"
	"github.com/billgraziano/xelogstash/pkg/log"
	"github.com/billgraziano/xelogstash/pkg/xe"
	"github.com/dustin/go-humanize"
	"github.com/dustin/go-humanize/english"
)

/*
-- server & session
*/
func main() {
	var server, session string
	var rows int
	var parse bool
	var allocs bool
	// var format bool
	flag.StringVar(&server, "server", "localhost", "server to connect to")
	flag.StringVar(&session, "session", "system_health", "extended event session name")
	flag.IntVar(&rows, "rows", 0, "maximum rows to read")
	flag.BoolVar(&parse, "parse", false, "parse the event XML data")
	flag.BoolVar(&allocs, "allocs", false, "print out the heap allocations")
	flag.Parse()
	log.Info(fmt.Sprintf("parameters: server: %s  session: %s", server, session))
	log.Info(fmt.Sprintf("parameters: rows: %s  parse: %v", humanize.Comma(int64(rows)), parse))
	run(server, session, rows, parse, false, allocs)
}

func run(server, session string, maxRows int, parse, format bool, allocs bool) {
	// 	mapping := `
	// 	root.mssql = this
	// 	root.mssql.test = "abc"
	// 	root.logstash_type = "orgsql"
	// 	root.timestamp = this.timestamp
	// 	root."@timestamp" = this.timestamp

	// 	root.orgsql.type = match this.name {
	// 		"login" => "mssql.login"
	// 		_ => "mssql.xe"
	// 	  }

	// 	root.id = ksuid()
	// 	root.ingest = now()
	// 	root.global.host.name = this.server_instance_name
	// 	# root.global.log.collector.host = hostname().lowercase()

	// 	#root.mssql.mssql_ag_listener = if this.exists("mssql_ag_listener") {
	// 	#	this.mssql_ag_listener.map_each(ag -> ag.uppercase())
	// 	#}
	// 	#root.received_at = timestamp_unix()

	// 	#root.mssql.duration_sec = if this.exists("duration") {
	// 	#	this.duration/1000000
	// 	# }
	// 	#root.global.host.org_division = "mtsus"
	// 	#root.mssql.new_value = 37
	// 	# root.mssql.static_dns = "sd-db-txn"
	// `

	// 	blobExec, err := bloblang.Parse(mapping)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}

	cxn := mssqlh.NewConnection(server, "", "", "master", "readfast.exe")
	info, err := xe.NewSQLInfo(cxn.Driver, cxn.String(), "", "")
	if err != nil {
		log.Fatal(err)
	}
	defer safeClose(info.DB, &err)
	log.Info(fmt.Sprintf("server:  %s (%s)", info.Server, info.Version))
	xeSession, err := xe.GetSession(info.DB, session)
	if err != nil {
		log.Fatal(err)
	}
	log.Info(fmt.Sprintf("session: %s (%s)", xeSession.Name, xeSession.WildCard))
	query := fmt.Sprintf("SELECT object_name, event_data, file_name, file_offset FROM sys.fn_xe_file_target_read_file('%s', NULL, NULL, NULL) OPTION (RECOMPILE);", xeSession.WildCard)
	start := time.Now()
	rows, err := info.DB.Query(query) // #nosec G201 -- string doeesn't come from user
	if err != nil {
		log.Fatal(err)
	}
	defer safeClose(rows, &err)

	var totalRows, fileRows int64
	var objectName, eventData, fileName string
	var fileOffset int64
	var savedFile string
	var fileCount int

	for rows.Next() {
		err = rows.Scan(&objectName, &eventData, &fileName, &fileOffset)
		if err != nil {
			log.Fatal(err)
		}
		if totalRows == 0 {
			log.Info(fmt.Sprintf("time to read first event: %s", time.Since(start).String()))
		}
		fileName = filepath.Base(fileName)
		if fileName != savedFile {
			if savedFile != "" {
				log.Info(fmt.Sprintf("file: %s  events: %s", filepath.Base(fileName), humanize.Comma(fileRows)))
			}
			savedFile = fileName
			fileRows = 0
			fileCount++
		}
		if parse {
			event, err := xe.Parse(&info, eventData, false)
			if err != nil {
				log.Fatal(err)
			}
			if fileRows == 0 {
				eventTime := event.Timestamp()
				log.Info(fmt.Sprintf("file: %s  first:  %s", fileName, eventTime.Format(time.RFC3339)))
			}
			// if format {
			// 	newObj, err := blobExec.Query(event)
			// 	if err != nil {
			// 		log.Error(fmt.Sprintf("EVENT: %v", event))
			// 		log.Fatal(err)
			// 	}
			// 	_, err = json.MarshalIndent(newObj, "", "  ")
			// 	if err != nil {
			// 		log.Fatal(err)
			// 	}
			// }

		} else {
			if fileRows == 0 {
				log.Info(fmt.Sprintf("file: %s", fileName))
			}
		}

		totalRows++
		fileRows++
		if maxRows > 0 && totalRows == int64(maxRows) {
			break
		}
	}
	log.Info(fmt.Sprintf("file: %s  events: %s", filepath.Base(fileName), humanize.Comma(fileRows)))
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	dur := time.Since(start)
	log.Info(fmt.Sprintf("total events: %s in %s (%s)", humanize.Comma(totalRows), dur.String(), english.Plural(int(fileCount), "file", "")))
	if dur > 1*time.Second {
		rowsPerMS := float64(totalRows) / float64(dur.Milliseconds())
		rowsPerSecond := rowsPerMS * 1000.0
		rowsPerMinute := rowsPerSecond * 60.0
		log.Info(fmt.Sprintf("events per second: %s", humanize.Commaf(math.Round(rowsPerSecond))))
		log.Info(fmt.Sprintf("events per minute: %s", humanize.Commaf(math.Round(rowsPerMinute))))
	}

	log.Info("done.")
	// runtime.GC()
	if allocs {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		fmt.Printf("Total allocations: %d bytes (%.2f MB)\n",
			m.TotalAlloc, float64(m.TotalAlloc)/(1024*1024))
		fmt.Printf("Mallocs: %d\n", m.Mallocs)
		fmt.Printf("alloc bytes/row: %d\n", m.TotalAlloc/uint64(totalRows))
		fmt.Printf("allocations/row: %d\n", m.Mallocs/uint64(totalRows))
	}
	/*
		D40\SQL2016 (system_health)
		alloc bytes/row: 24794
		allocations/row: 372

		D40\SQL2016 (logstgash_events)
		alloc bytes/row: 31181
		allocations/row: 371
	*/
}

func safeClose(c io.Closer, err *error) {
	cerr := c.Close()
	if cerr != nil {
		log.Error("safeClose: ", cerr)
		if *err == nil {
			*err = cerr
		}
	}
}
