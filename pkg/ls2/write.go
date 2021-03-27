package ls2

import (
	"fmt"
	"time"

	"github.com/lestrrat-go/backoff"
	"github.com/pkg/errors"
)

func (ls *Logstash) Write(message string) error {
	var err error
	ls.RLock()
	err = ls.write(message)
	ls.RUnlock()
	if err == nil {
		return nil
	}
	//ls.Logger.Error(errors.Wrap(err, "ls2: write"))

	// do the retry stuff
	ls.Lock()
	defer ls.Unlock()
	return ls.writeloop(message)
}

func (ls *Logstash) write(message string) error {
	var err error
	if ls.Connection == nil {
		return errors.New("nil connection")
	}
	message = fmt.Sprintf("%s\n", message)
	messageBytes := []byte(message)
	if trace {
		ls.Logger.Tracef("ls2.write.bytes-to-send: %d", len(messageBytes))
	}

	deadline := time.Now().Add(time.Duration(ls.Timeout) * time.Second)
	ls.Connection.SetDeadline(deadline)
	var n int
	n, err = ls.Connection.Write(messageBytes)
	if err != nil {
		return errors.Wrap(err, "ls.connection.write")
	}
	if n != len(messageBytes) {
		return errors.Errorf("ls.connection.write: bytes: %d; sent: %d", len(messageBytes), n)
	}

	return nil
}

func (ls *Logstash) writeloop(message string) error {
	var err error
	// first, try the write again in case someone else fixed it
	err = ls.write(message)
	if err == nil {
		return nil
	}

	ls.Logger.Error(errors.Wrap(err, "writeloop: ls.write"))

	var policy = backoff.NewExponential(
		backoff.WithInterval(10*time.Second), // base interval
		backoff.WithJitterFactor(0.10),       // 10% jitter
		backoff.WithMaxInterval(120*time.Second),
		backoff.WithMaxRetries(0),
	)

	i := 0
	bo, cancel := policy.Start(ls.ctx)
	defer cancel()
	for backoff.Continue(bo) {
		i++
		ls.Logger.Errorf("writeloop: retrying %s (#%d)", ls.Host, i)

		err = ls.close()
		if err != nil {
			ls.Logger.Error(errors.Wrap(err, "writeloop: ls.close"))
			continue
		}
		// open()
		err = ls.open()
		if err != nil {
			ls.Logger.Error(errors.Wrap(err, "writeloop: ls.open"))
			continue
		}
		// write()
		err = ls.write(message)
		if err != nil {
			ls.Logger.Error(errors.Wrap(err, "writeloop: ls.write"))
		} else { // write with no error means we break out of the loop
			ls.Logger.Infof("writeloop: write succeeded: %s", ls.Host)
			break
		}

	}
	return nil
}
