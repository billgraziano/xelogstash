package sink

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/billgraziano/xelogstash/logstash"
	"github.com/lestrrat-go/backoff"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// LogstashSink write events to logstash
type LogstashSink struct {
	mu     sync.RWMutex
	ls     *logstash.Logstash
	logger *log.Entry
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
	return lss.name()
}

func (lss *LogstashSink) name() string {
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
func (lss *LogstashSink) Open(ctx context.Context, ignore string) error {
	var err error
	lss.mu.Lock()
	defer lss.mu.Unlock()
	err = lss.open(ignore)
	if err == nil {
		return nil
	}

	time.Sleep(1 * time.Second)

	// start retry logic
	var policy = backoff.NewExponential(
		backoff.WithInterval(1*time.Second), // base interval
		backoff.WithJitterFactor(0.10),      // 10% jitter
		backoff.WithMaxInterval(30*time.Second),
		backoff.WithMaxRetries(0),
	)

	i := 1
	bo, cancel := policy.Start(ctx)
	defer cancel()
	for backoff.Continue(bo) {

		if lss.logger != nil {
			log.Errorf("open error: retrying %s (#%d) [%s]...", lss.name(), i, err.Error())
		}
		err = lss.open(ignore)
		if err == nil {
			log.Infof("open succeeded: %s", lss.name())
			return nil
		}

		i++
	}
	return errors.Wrap(err, "logstash.open")
}

func (lss *LogstashSink) open(ignore string) error {
	_, err := lss.ls.Connect()
	if err != nil {
		return errors.Wrap(err, "logstash.connect")
	}
	return nil
}

func (lss *LogstashSink) Write(ctx context.Context, name, event string) (int, error) {
	var err error
	var n int

	// first just try to lock on the happy path
	lss.mu.RLock()
	n, err = lss.write(name, event)
	lss.mu.RUnlock()
	if err == nil {
		return n, err
	}

	time.Sleep(1 * time.Second)
	// write failed, start retry logic
	var policy = backoff.NewExponential(
		backoff.WithInterval(1*time.Second), // base interval
		backoff.WithJitterFactor(0.10),      // 10% jitter
		backoff.WithMaxRetries(10),          // If not specified, default number of retries is 10
		backoff.WithMaxInterval(30*time.Second),
	)

	i := 1
	bo, cancel := policy.Start(ctx)
	defer cancel()
	for backoff.Continue(bo) {

		if lss.logger != nil {
			log.Errorf("write error: retrying %s (#%d) [%s]...", lss.name(), i, err.Error())
		}
		lss.mu.Lock()
		// ignore errors, we get those if it down
		xerr := lss.reopen()
		if xerr != nil {
			log.Error(errors.Wrap(xerr, "lss.reopen"))
		}
		lss.mu.Unlock()

		lss.mu.RLock()
		n, err = lss.write(name, event)
		lss.mu.RUnlock()
		if err == nil {
			log.Infof("write succeeded: %s", lss.name())
			return n, nil
		}
		i++
	}
	return n, errors.Wrap(err, "logstash.write")
}

func (lss *LogstashSink) write(name, event string) (int, error) {
	err := lss.ls.Writeln(event)
	if err != nil {
		return 0, errors.Wrap(err, "logstash.writeln")
	}
	return 0, nil
}

// Reopen closes and reopens a TCP connection.
// Typically used for an error condition or a dynamic IP changed
func (lss *LogstashSink) Reopen() error {
	lss.mu.Lock()
	defer lss.mu.Unlock()
	err := lss.reopen()
	if err != nil {
		return errors.Wrap(err, "lss.reopen")
	}
	return nil
}

func (lss *LogstashSink) reopen() error {
	if lss.logger != nil {
		log.Trace("reopening...")
	}
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

// SetLogger sets the logger for the sink
func (lss *LogstashSink) SetLogger(entry *log.Entry) {
	lss.logger = entry
}
