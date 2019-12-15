package logstash

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

// Severity is the severity for a record
type Severity int

// use to trace writing bytes in DEV build
var trace bool = false

const (
	// Error event
	Error Severity = 3
	// Warning event
	Warning Severity = 4
	// Info event
	Info Severity = 6
)

// Logstash is the basic struct
type Logstash struct {
	Connection *net.TCPConn
	Timeout    int    //Timeout in seconds
	Host       string // Host in host:port format
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

// NewHost generates a logstash sender from a host:port format
func NewHost(host string, timeout int) (*Logstash, error) {

	var err error
	ls := &Logstash{}

	_, lsportstring, err := net.SplitHostPort(host)
	if err != nil {
		return ls, errors.Wrap(err, "net-splithost")
	}
	_, err = strconv.Atoi(lsportstring)
	if err != nil {
		return ls, errors.Wrap(err, "logstash port isn't numeric")
	}

	ls.Host = host
	ls.Timeout = timeout

	return ls, nil
}

// SetTimeouts sets the TCPConn timeout value from the LogStash object
func (ls *Logstash) setTimeouts() {
	deadline := time.Now().Add(time.Duration(ls.Timeout) * time.Second)
	ls.Connection.SetDeadline(deadline)
}

// Close the underlying TCP connection
func (ls *Logstash) Close() error {
	if ls.Connection != nil {
		return ls.Connection.Close()
	}
	return nil
}

// Connect to the host
func (ls *Logstash) Connect() (*net.TCPConn, error) {
	var connection *net.TCPConn
	addr, err := net.ResolveTCPAddr("tcp", ls.Host)
	if err != nil {
		return connection, errors.Wrap(err, "net.resolveicpaddr")
	}
	connection, err = net.DialTCP("tcp", nil, addr)
	if err != nil {
		return connection, errors.Wrap(err, "net.dialtcp")
	}
	if connection != nil {
		ls.Connection = connection
		//ls.Connection.SetLinger(0)
		ls.Connection.SetKeepAlive(true)
		//ls.Connection.SetKeepAlivePeriod(time.Duration(60) * time.Second)
		//ls.setTimeouts()
	}
	if connection == nil && err == nil {
		return connection, errors.New("conn & err can't both be nil")
	}
	return connection, err
}

// Writeln send a message to the host
func (ls *Logstash) Writeln(message string) error {

	var err error
	if ls.Connection == nil {
		_, err = ls.Connect()
		if err != nil {
			return errors.Wrap(err, "connect")
		}
	}

	message = fmt.Sprintf("%s\n", message)
	messageBytes := []byte(message)
	if trace {
		fmt.Println(fmt.Sprintf("ls.writeln.bytes-to-send: %d", len(messageBytes)))
	}

	var n int
	ls.setTimeouts()
	n, err = ls.Connection.Write(messageBytes)
	if trace {
		fmt.Println(fmt.Sprintf("ls.connection.write.bytes-sent: %d", n))
	}
	if n != len(messageBytes) {
		fmt.Printf("send bytes mismatch: wanted: %d; sent: %d\r\n", len(messageBytes), n)
	}
	if err != nil {
		if trace {
			fmt.Println(fmt.Sprintf("ls.connection.write.err: %s", err.Error()))
		}
		neterr, ok := err.(net.Error)
		if ok && neterr.Timeout() {
			ls.Connection.Close()
			ls.Connection = nil
			if err != nil {
				return errors.Wrap(err, "write-timeout")
			}
		} else {
			ls.Connection.Close()
			ls.Connection = nil
			return errors.Wrap(err, "write")
		}

		if trace {
			fmt.Println("ls.connection.write: success-inside")
		}
		return nil
	}
	if trace {
		fmt.Println("ls.connection.write: fall-through")
	}
	return nil
}
