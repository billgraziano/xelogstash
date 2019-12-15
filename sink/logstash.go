package sink

import (
	"fmt"
	"sync"

	"github.com/billgraziano/xelogstash/logstash"
	"github.com/pkg/errors"
)

// LogstashSink write events to logstash
type LogstashSink struct {
	ls *logstash.Logstash
	mu sync.RWMutex
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
	lss.mu.RLock()
	defer lss.mu.RUnlock()
	if lss != nil {
		return fmt.Sprintf("logstash: %s", lss.ls.Host)
	}
	return ""

}

// Close a LogstashSink
func (lss *LogstashSink) Close() error {
	if lss.ls == nil {
		return nil
	}
	lss.mu.Lock()
	defer lss.mu.Unlock()
	return lss.close()
}

func (lss *LogstashSink) close() error {
	if lss.ls == nil {
		return nil
	}
	if lss.ls.Connection == nil {
		return nil
	}
	err := lss.ls.Connection.Close()
	if err != nil {
		return errors.Wrap(err, "lss.ls.close")
	}
	return nil
}

// Open a LogstashSink
func (lss *LogstashSink) Open(ignore string) error {
	lss.mu.Lock()
	defer lss.mu.Unlock()
	return lss.open(ignore)
}

func (lss *LogstashSink) open(ignore string) error {
	_, err := lss.ls.Connect()
	if err != nil {
		return errors.Wrap(err, "logstash.connect")
	}
	return nil
}

// Write an event to the sink
func (lss *LogstashSink) Write(name, event string) (int, error) {
	lss.mu.RLock()
	defer lss.mu.RUnlock()
	return lss.write(name, event)
}

func (lss *LogstashSink) write(name, event string) (int, error) {
	// TODO Backoff
	// https://github.com/lestrrat-go/backoff,
	// https://github.com/cenkalti/backoff
	err := lss.ls.Writeln(event)
	if err != nil {
		return 0, errors.Wrap(err, "logstash.writeln")
	}
	return 0, nil
}

// Reopen closes and reopens a TCP connection.
// Typically used for an error condition or a dynamic IP changed
func (lss *LogstashSink) Reopen() error {
	println("reopening...")
	lss.mu.Lock()
	defer lss.mu.Unlock()
	err := lss.close()
	if err != nil {
		return errors.Wrap(err, "lss.close")
	}
	err = lss.open("")
	if err != nil {
		return errors.Wrap(err, "lss.open")
	}
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
