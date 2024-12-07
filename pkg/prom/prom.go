package prom

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	EventsRead = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sqlxewriter_event_read_total",
			Help: "Total number of extended events read from SQL Server",
		},
		[]string{"event", "server"},
	)

	EventsWritten = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sqlxewriter_event_write_total",
			Help: "Total number of extended events written to a sink",
		},
		[]string{"event", "server"},
	)

	BytesWritten = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sqlxewriter_event_write_bytes",
			Help: "Total bytes of JSON written to a sink",
		},
		[]string{"event", "server"},
	)
)

func init() {
	prometheus.MustRegister(EventsRead)
	prometheus.MustRegister(EventsWritten)
	prometheus.MustRegister(BytesWritten)
}

// ServerLabel returns a string in the format
// domain_computer[_instance] in lower case
func ServerLabel(domain, server string) string {
	var label string
	if domain != "" {
		label += domain + "_"
	}
	if server != "" {
		label += strings.Join(strings.Split(server, "\\"), "_")
	}
	return strings.ToLower(label)
}
