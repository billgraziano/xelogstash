package ls2

import (
	"net"
	"time"

	"github.com/lestrrat-go/backoff"
	"github.com/pkg/errors"
)

func (ls *Logstash) Connect() error {
	var err error
	ls.Lock()
	defer ls.Unlock()

	// if there's a connection, close it
	if ls.Connection != nil {
		// just log the close error
		err = ls.close()
		if err != nil {
			ls.Logger.Errorf("ls2: connect: close: %s (ignored)", err.Error())
		}
	}

	err = ls.connect()
	if err != nil {
		return errors.Wrap(err, "ls2.connect")
	}
	return nil
}

func (ls *Logstash) connect() error {

	var err error
	// first try to open
	err = ls.open()
	if err == nil {
		return nil
	}

	// then run the back off policy
	var policy = backoff.NewExponential(
		backoff.WithInterval(10*time.Second), // base interval
		backoff.WithJitterFactor(0.10),       // 10% jitter
		backoff.WithMaxInterval(120*time.Second),
		backoff.WithMaxRetries(0),
	)

	i := 1
	bo, cancel := policy.Start(ls.ctx)
	defer cancel()
	for backoff.Continue(bo) {
		ls.Logger.Errorf("ls2: open error: retrying %s (#%d) [%s]...", ls.Host, i, err.Error())
		err = ls.open()
		if err == nil {
			ls.Logger.Infof("ls2: open succeeded: %s", ls.Host)
			return nil
		}
		// Do I need to throw away any partial connections?
		i++
	}
	return errors.Wrap(err, "ls2.connect")
}

func (ls *Logstash) open() error {
	var cxn *net.TCPConn
	addr, err := net.ResolveTCPAddr("tcp", ls.Host)
	if err != nil {
		return errors.Wrap(err, "net.resolvetcpaddr")
	}
	ls.Logger.Debugf("ls2: host: %s; addr: %s", ls.Host, addr)
	cxn, err = net.DialTCP("tcp", nil, addr)
	if err != nil {
		return errors.Wrap(err, "net.dialtcp")
	}
	if cxn == nil {
		return errors.New("nil connection")
	}
	ls.Connection = cxn
	ls.Connection.SetKeepAlive(true)

	return nil
}
