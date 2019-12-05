package main

import (
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// func safeClose(c io.Closer, err *error) {
// 	cerr := c.Close()
// 	if cerr != nil {
// 		log.Error("safeClose: ", cerr)
// 		if *err == nil {
// 			*err = cerr
// 		}
// 	}
// }

// // writes a single message to logstash
// func writeToLogstash(addr *net.TCPAddr, message string) error {
// 	var err error

// 	c, err := net.DialTCP("tcp", nil, addr)
// 	if err != nil {
// 		return errors.Wrap(err, "net.DialTCP")
// 	}
// 	defer safeClose(c, &err)

// 	//var msg []byte
// 	// if opts.Test {
// 	// 	msg = []byte("{}")
// 	// } else {
// 	// 	msg = []byte(message)
// 	// }
// 	msg := []byte(message)

// 	_, err = c.Write(msg)
// 	if err != nil {
// 		return errors.Wrap(err, "conn.write")
// 	}
// 	return nil
// }

// resolves a host name and port to an IP address and port
func resolveIP(hostAndPort string) (addr *net.TCPAddr, err error) {
	addr, err = net.ResolveTCPAddr("tcp", hostAndPort)
	if err != nil {
		return addr, errors.Wrap(err, "resolveTCPAddr")
	}
	return addr, nil
}

func containsString(array []string, search string) bool {
	s := strings.ToLower(search)
	for _, v := range array {
		if s == strings.ToLower(v) {
			return true
		}
	}
	return false
}

func getDefaultConfigFileName() (fn string, err error) {
	var s string
	s, err = os.Executable()
	if err != nil {
		return "", errors.Wrap(err, "os.executable")
	}
	s = filepath.Base(s)
	base := strings.TrimSuffix(s, path.Ext(s))
	fn = base + ".toml"

	return fn, nil
}
