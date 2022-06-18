package sink

import (
	"context"
	"fmt"

	"github.com/billgraziano/xelogstash/pkg/ls2"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Note: All locking moved to ls2

// LogstashSink write events to logstash
type LogstashSink struct {
	ls     *ls2.Logstash
	logger *log.Entry
}

// NewLogstashSink returns a new LogstashSink
func NewLogstashSink(host string, timeout int) (*LogstashSink, error) {
	ls, err := ls2.NewHost(host, ls2.WithTimeout(timeout))
	if err != nil {
		return nil, errors.Wrap(err, "ls2.newhost")
	}
	lss := &LogstashSink{
		ls:     ls,
		logger: log.WithFields(log.Fields{}),
	}

	return lss, nil
}

// Name returns the name of logstash
func (lss *LogstashSink) Name() string {
	return lss.name()
}

func (lss *LogstashSink) name() string {
	if lss != nil {
		return fmt.Sprintf("logstashsink: host: %s", lss.ls.Host)
	}
	return "unknown"
}

// Close a LogstashSink
func (lss *LogstashSink) Close() error {
	return lss.ls.Close()
}

// Open a LogstashSink
func (lss *LogstashSink) Open(ctx context.Context, ignore string) error {
	return lss.ls.Connect()
}

func (lss *LogstashSink) Write(ctx context.Context, name, event string) (int, error) {
	return 0, lss.ls.Write(event)
}

// Reopen is a noop -- handled in the ls2 package
func (lss *LogstashSink) Reopen() error {
	return nil
}

// Flush is a noop for logstash
func (lss *LogstashSink) Flush() error {
	return nil
}

// Clean is a noop for logstash
func (lss *LogstashSink) Clean() error {
	return nil
}

// SetLogger sets the logger for the sink
func (lss *LogstashSink) SetLogger(entry *log.Entry) {
	lss.logger = entry
	lss.ls.Logger = entry
}
