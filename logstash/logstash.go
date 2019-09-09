package logstash

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/pkg/errors"
)

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

// Logstash is the basic struct
type Logstash struct {
	Connection *net.TCPConn
	Timeout    int    //Timeout in seconds
	Host       string // Host in host:port format
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
		ls.Connection.SetLinger(0)
		ls.Connection.SetKeepAlive(true)
		//ls.Connection.SetKeepAlivePeriod(time.Duration(60) * time.Second)
		ls.setTimeouts()
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

	_, err = ls.Connection.Write([]byte(message))
	if err != nil {
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

		// Successful write! Let's extend the timeoul.
		ls.setTimeouts()
		return nil
	}

	return err
}

// Record holds the parent struct of what we will send to logstash
type Record map[string]interface{}

// NewRecord initializes a new record
func NewRecord() Record {
	r := make(map[string]interface{})
	return r
}

// ToLower sets most fields to lower case.  Fields like message
// and various SQL statements are unchanged
// func (e *Record) ToLower() {
// 	for k, v := range *e {
// 		if k != "message" && k != "timestamp" && k != "sql_text" && k != "statement" && k != "batch_text" {
// 			s, ok := v.(string)
// 			if ok {
// 				(*e)[k] = strings.ToLower(s)
// 			}
// 		}
// 	}
// }

// ToJSON marshalls to a string
func (r *Record) ToJSON() (string, error) {
	jsonBytes, err := json.Marshal(r)
	if err != nil {
		return "", errors.Wrap(err, "marshal")
	}

	jsonString := string(jsonBytes)
	return jsonString, nil
}

// ToJSONBytes marshalls to a byte array
func (r *Record) ToJSONBytes() ([]byte, error) {
	jsonBytes, err := json.Marshal(r)
	if err != nil {
		return []byte{}, errors.Wrap(err, "marshal")
	}
	return jsonBytes, nil
}

// ProcessMods applies adds, renames, and moves to a JSON string
func ProcessMods(json string, adds, copies, moves map[string]string) (string, error) {
	var err error

	// Adds
	for k, v := range adds {
		i := getValue(v)
		if gjson.Get(json, k).Exists() {
			return json, errors.Wrapf(err, "can't overwrite key: %s", k)
		}
		json, err = sjson.Set(json, k, i)
		if err != nil {
			return json, errors.Wrapf(err, "sjson.set: %s %s", k, v)
		}
	}

	// Copies
	for src, dst := range copies {

		if gjson.Get(json, dst).Exists() {
			return json, errors.Wrapf(err, "can't overwrite key: %s", dst)
		}
		r := gjson.Get(json, src)
		if !r.Exists() {
			continue
		}
		json, err = sjson.Set(json, dst, doubleSlashes(r.Value()))
		if err != nil {
			return json, errors.Wrapf(err, "sjson.set: %s %v", dst, r.Value())
		}
		//fmt.Println(r.Value(), doubleSlashes(r.Value()))
	}

	// Moves
	for src, dst := range moves {

		if gjson.Get(json, dst).Exists() {
			return json, errors.Wrapf(err, "can't overwrite key: %s", dst)
		}
		r := gjson.Get(json, src)
		if !r.Exists() {
			continue
		}
		json, err = sjson.Set(json, dst, doubleSlashes(r.Value()))
		if err != nil {
			return json, errors.Wrapf(err, "sjson.set: %s %v", dst, r.Value)
		}
		json, err = sjson.Delete(json, src)
		if err != nil {
			return json, errors.Wrapf(err, "can't delete: %s", src)
		}
	}

	return json, err
}

func doubleSlashes(v interface{}) interface{} {
	x, ok := v.(string)
	if !ok {
		return v
	}
	return strings.Replace(x, "\\", "\\\\", -1)
}

func getValue(s string) (v interface{}) {
	var err error
	v, err = strconv.ParseBool(s)
	if err == nil {
		return v
	}

	v, err = strconv.ParseInt(s, 0, 64)
	if err == nil {
		return v
	}

	v, err = strconv.ParseFloat(s, 64)
	if err == nil {
		return v
	}

	// check for '0.7' => (string) 0.7
	if len(s) >= 2 && strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") {
		s = s[1 : len(s)-1]
	}

	return doubleSlashes(s)
}

// Set assigns a string value to a key in the event
func (r *Record) Set(key string, value interface{}) {
	(*r)[key] = value
}

// Copy value from srckey to newkey
func (r *Record) Copy(srckey, newkey string) {
	v, ok := (*r)[srckey]
	if !ok {
		r.Set(newkey, "")
		return
	}
	(*r)[newkey] = v
}

// Move old key to new key
func (r *Record) Move(oldkey, newkey string) {
	(*r).Copy(oldkey, newkey)
	delete((*r), oldkey)
}

// SetIfEmpty sets a value if one doesn't already exist
func (r *Record) SetIfEmpty(key string, value interface{}) {
	_, exists := (*r)[key]
	if !exists {
		r.Set(key, value)
	}
}
