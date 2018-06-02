package main

import (
	"database/sql"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/billgraziano/xelogstash/log"
	"github.com/pkg/errors"
)

// SQLInfo stores details about the server we connected to
type SQLInfo struct {
	Server   string
	Domain   string
	Computer string
	// FQDN           string
	ProductLevel   string
	ProductRelease string
	Version        string
}

// GetInstance returns the instance and domain name
func GetInstance(db *sql.DB, session string) (info SQLInfo, err error) {

	query := `
	SET NOCOUNT ON;
	SELECT	@@SERVERNAME AS [ServerName]
		,DEFAULT_DOMAIN() AS [DomainName]
		,CAST(SERVERPROPERTY('MachineName') as nvarchar(128)) AS [Computer]
		,CAST(COALESCE(SERVERPROPERTY('ProductLevel'), '') as nvarchar(128)) AS ProductLevel
		,COALESCE(CAST(SERVERPROPERTY('ProductMajorVersion') as NVARCHAR(128))  + '.' + CAST(SERVERPROPERTY('ProductMinorVersion') as NVARCHAR(128)),'') AS ProductRelease
	`
	row := db.QueryRow(query)
	err = row.Scan(&info.Server, &info.Domain, &info.Computer, &info.ProductLevel, &info.ProductRelease)
	if err != nil {
		return info, errors.Wrap(err, "scan")
	}
	var v string
	switch info.ProductRelease {
	case "15.0":
		v = "SQL Server vNext"
	case "14.0":
		v = "SQL Server 2017"
	case "13.0":
		v = "SQL Server 2016"
	case "12.0":
		v = "SQL Server 2014"
	case "11.0":
		v = "SQL Server 2012"
	case "10.5":
		v = "SQL Server 2008 R2"
	case "10.0":
		v = "SQL Server 2008"
	case "9.0":
		v = "SQL Server 2005"
	default:
		v = "unknown"
	}
	info.Version = fmt.Sprintf("%s %s", v, info.ProductLevel)

	return info, nil
}

func safeClose(c io.Closer, err *error) {
	cerr := c.Close()
	if cerr != nil {
		log.Error("safeClose: ", cerr)
		if *err == nil {
			*err = cerr
		}
	}
}

// writes a single message to logstash
func writeToLogstash(addr *net.TCPAddr, message string) error {
	var err error

	c, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return errors.Wrap(err, "net.DialTCP")
	}
	defer safeClose(c, &err)

	//var msg []byte
	// if opts.Test {
	// 	msg = []byte("{}")
	// } else {
	// 	msg = []byte(message)
	// }
	msg := []byte(message)

	_, err = c.Write(msg)
	if err != nil {
		return errors.Wrap(err, "conn.write")
	}
	return nil
}

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

func roundDuration(d, r time.Duration) time.Duration {
	if r <= 0 {
		return d
	}
	neg := d < 0
	if neg {
		d = -d
	}
	if m := d % r; m+m < r {
		d = d - m
	} else {
		d = d + r - m
	}
	if neg {
		return -d
	}
	return d
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
