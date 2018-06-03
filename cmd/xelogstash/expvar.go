package main

import "expvar"

var (
	totalCount  = expvar.NewInt("totalEvents")
	eventCount  = expvar.NewMap("events").Init()
	serverCount = expvar.NewMap("servers").Init()
)
