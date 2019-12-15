package sink

import (
	"context"

	log "github.com/sirupsen/logrus"
)

// Sinker defines places that events can be written
type Sinker interface {
	Open(context.Context, string) error
	Write(context.Context, string, string) (int, error)
	Flush() error
	Close() error
	Name() string
	Clean() error
	Reopen() error
	SetLogger(*log.Entry)
}

// * filesink: file_name
// * logstash: host & port
