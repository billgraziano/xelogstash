package app

import (
	"expvar"
	"runtime"
	"time"

	"github.com/billgraziano/xelogstash/pkg/metric"
)

var (
	totalCount  = expvar.NewInt("eventsProcessed")
	eventCount  = expvar.NewMap("events").Init()
	serverCount = expvar.NewMap("servers").Init()
	readCount   = expvar.NewInt("eventsRead")
	//expWorker   = expvar.NewMap("workers").Init()
)

// ConfigureExpvar sets up the expvar variables
func ConfigureExpvar() {

	// configure monitoring
	expvar.Publish("app:eventsRead", metric.NewCounter("5m10s", "60m1m", "24h15m"))
	expvar.Publish("app:eventsWritten", metric.NewCounter("5m10s", "60m1m", "24h15m"))
	expvar.Publish("go:alloc", metric.NewGauge("60m15s", "48h15m"))
	expvar.Publish("go:numgoroutine", metric.NewGauge("60m15s", "48h15m"))
	expvar.Publish("go:sys", metric.NewGauge("60m15s", "48h15m"))

	m := &runtime.MemStats{}
	runtime.ReadMemStats(m)
	expvar.Get("go:numgoroutine").(metric.Metric).Add(float64(runtime.NumGoroutine()))
	expvar.Get("go:alloc").(metric.Metric).Add(float64(m.Alloc) / 1000000)
	expvar.Get("go:sys").(metric.Metric).Add(float64(m.Sys) / 1000000)

	go func() {
		for range time.Tick(60 * time.Second) {
			m := &runtime.MemStats{}
			runtime.ReadMemStats(m)
			expvar.Get("go:numgoroutine").(metric.Metric).Add(float64(runtime.NumGoroutine()))
			expvar.Get("go:alloc").(metric.Metric).Add(float64(m.Alloc) / 1000000)
			expvar.Get("go:sys").(metric.Metric).Add(float64(m.Sys) / 1000000)
		}
	}()
}
