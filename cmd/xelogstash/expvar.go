package main

import "expvar"

var (
	totalCount *expvar.Int
	eventCount = expvar.NewMap("events").Init()
)

func init() {
	totalCount = expvar.NewInt("totalEvents")
}
