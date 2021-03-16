package ls2

import (
	"context"
	"net"
	"strconv"
	"sync"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// use to trace writing bytes in DEV build
var trace bool = false

// Severity is the severity for a record
type Severity int

const (
	// Error event
	Error Severity = 3
	// Warning event
	Warning Severity = 4
	// Info event
	Info Severity = 6
)

// Option configures a Logstash host
type Option func(l *Logstash)

// Logstash is the basic struct
type Logstash struct {
	sync.RWMutex
	Connection *net.TCPConn
	Timeout    int    //Timeout in seconds
	Host       string // Host in host:port format
	Logger     *log.Entry
	ctx        context.Context
}

// NewHost returns a Logstash
func NewHost(host string, opts ...func(*Logstash)) (*Logstash, error) {
	var err error

	_, lsportstring, err := net.SplitHostPort(host)
	if err != nil {
		return &Logstash{}, errors.Wrap(err, "want host:port")
	}
	_, err = strconv.Atoi(lsportstring)
	if err != nil {
		return &Logstash{}, errors.Wrap(err, "logstash port isn't numeric")
	}

	ls := &Logstash{
		RWMutex: sync.RWMutex{},
		Host:    host,
		Timeout: 60,
		ctx:     context.Background(),
		Logger:  log.WithFields(log.Fields{}),
	}

	for _, opt := range opts {
		opt(ls)
	}

	return ls, nil
}

// WithContext configures context
func WithContext(ctx context.Context) Option {
	return func(l *Logstash) {
		l.ctx = ctx
	}
}

// WithTimeout configures a timeout
func WithTimeout(timeout int) Option {
	return func(l *Logstash) {
		l.Timeout = timeout
	}
}

// WithLogger configures a timeout
func WithLogger(le *log.Entry) Option {
	return func(l *Logstash) {
		l.Logger = le
	}
}

func (s Severity) String() string {
	switch s {
	case 3:
		return "err"
	case 4:
		return "warning"
	case 6:
		return "info"
	default:
		return "info"
	}
}
