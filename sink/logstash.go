package sink

import (
	"fmt"

	"github.com/billgraziano/xelogstash/logstash"
	"github.com/pkg/errors"
)

// LogstashSink write events to logstash
type LogstashSink struct {
	ls *logstash.Logstash
}

// NewLogstashSink returns a new LogstashSink
func NewLogstashSink(host string, timeout int) (*LogstashSink, error) {
	ls, err := logstash.NewHost(host, timeout)
	if err != nil {
		return nil, errors.Wrap(err, "logstash.newhost")
	}
	lss := &LogstashSink{ls: ls}
	return lss, nil
}

// Name returns the name of logstash
func (lss *LogstashSink) Name() string {
	return fmt.Sprintf("logstash: %s", lss.ls.Host)
}

// Close a LogstashSink
func (lss *LogstashSink) Close() error {
	return lss.ls.Close()
}

// Open a LogstashSink
func (lss *LogstashSink) Open(ignore string) error {
	_, err := lss.ls.Connect()
	if err != nil {
		return errors.Wrap(err, "logstash.connect")
	}
	return nil
}

// Write an event to the sink
func (lss *LogstashSink) Write(name, event string) (int, error) {
	// TODO Backoff
	// https://github.com/lestrrat-go/backoff,
	// https://github.com/cenkalti/backoff
	err := lss.ls.Writeln(event)
	if err != nil {
		return 0, errors.Wrap(err, "logstash.writeln")
	}
	return 0, nil
}

// Flush is a noop for logstash
func (lss *LogstashSink) Flush() error {
	return nil
}

// Clean is a noop for logstash
func (lss *LogstashSink) Clean() error {
	return nil
}
