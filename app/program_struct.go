package app

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/billgraziano/xelogstash/config"

	"github.com/billgraziano/xelogstash/sink"
	log "github.com/sirupsen/logrus"
)

// Program holds the program configuration
type Program struct {
	SHA1      string
	Version   string
	StartTime time.Time
	Verbose   bool // enable a little logging at the INFO level
	LogLevel  log.Level
	Loop      bool // run in a loop
	Server    *http.Server

	wg          sync.WaitGroup
	Cancel      context.CancelFunc
	WatchConfig bool
	//FSCancel context.CancelFunc
	targets int

	// Lock so we can handle file changes
	sync.Mutex

	// Interval that we poll servers in seconds
	// PollInterval int

	// ExtraDelay that is added to for testing
	// the stop function (in seconds)
	// This will probably be removed.
	ExtraDelay int

	// If we're writing to a file sink, which is an event rotator,
	// this has a pointer to it so we can close it at the very end
	// eventRotator *sink.Rotator

	Sinks []*sink.Sinker

	Filters []config.Filter
}
