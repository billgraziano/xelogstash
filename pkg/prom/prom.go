package prom

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

// TODO labels: event, domain, server (computer__instance), computer, instance

var (
	EventsRead = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sqlxewriter_event_read_total",
			Help: "Total number of extended events read from SQL Server",
		},
		[]string{"event", "domain", "server"},
	)

	EventsWritten = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sqlxewriter_event_write_total",
			Help: "Total number of extended events written to a sink",
		},
		[]string{"event", "domain", "server"},
	)

	BytesWritten = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sqlxewriter_event_write_bytes",
			Help: "Total bytes of JSON written to a sink",
		},
		[]string{"event", "domain", "server"},
	)
)

func init() {
	prometheus.MustRegister(EventsRead)
	prometheus.MustRegister(EventsWritten)
	prometheus.MustRegister(BytesWritten)
}

// ServerLabel accepts @@SERVERNAME in COMPUTER[\\INSTANCE]
// and returns computer[__instance] in lower case
func ServerLabel(server string) string {
	var label string

	if server != "" {
		label += strings.Join(strings.Split(server, "\\"), "__")
	}
	return strings.ToLower(label)
}
