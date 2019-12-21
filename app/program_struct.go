package app

import (
	"context"
	"sync"
	"time"

	"github.com/billgraziano/xelogstash/sink"
)

type Program struct {
	SHA1      string
	Version   string
	StartTime time.Time
	Verbose   bool // enable a little logging at the INFO level

	//Debug bool

	wg      sync.WaitGroup
	Cancel  context.CancelFunc
	targets int

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
}
