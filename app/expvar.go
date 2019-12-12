package app

import (
	"expvar"
)

var (
	totalCount  = expvar.NewInt("eventsProcessed")
	eventCount  = expvar.NewMap("events").Init()
	serverCount = expvar.NewMap("servers").Init()
	readCount   = expvar.NewInt("eventsRead")
	//expWorker   = expvar.NewMap("workers").Init()
)

func init() {

}
