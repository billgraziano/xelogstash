package mssqlodbc

import (
	"fmt"

	"github.com/pkg/errors"
	"strings"
)

// Connection holds information about an ODBC SQL Server connection
type Connection struct {
	driver              string
	Server              string
	User                string
	Password            string
	Trusted             bool
	AppName             string
	Database            string
	MultiSubnetFailover bool
}

// Driver gets the driver for the connection
func (c *Connection) Driver() string {
	return c.driver
}

// SetDriver sets the driver for a connection
func (c *Connection) SetDriver(d string) error {
	err := ValidDriver(d)
	if err != nil {
		return err
	}
	c.driver = d
	return nil
}

// ConnectionString returns a connection string
func (c *Connection) ConnectionString() (string, error) {

	// https://docs.microsoft.com/en-us/sql/relational-databases/native-client/applications/using-connection-string-keywords-with-sql-server-native-client

	var cxn string

	if c.driver == "" {
		driver, err := BestDriver()
		if err == ErrNoDrivers {
			return "", err
		}
		if err != nil {
			return "", errors.Wrap(err, "bestdriver")
		}
		c.driver = driver
	}

	// Driver
	cxn += fmt.Sprintf("Driver={%s}; ", c.driver)

	// Host
	// {SQL Server needs Server so we'll use this as default}
	if c.Server == "" {
		return "", errors.New("invalid server")
	}
	cxn += fmt.Sprintf("Server=%s; ", c.Server)

	// Authentication
	if c.Trusted || (c.User == "" && c.Password == "") {
		cxn += fmt.Sprintf("Trusted_Connection=Yes; ")
	} else {
		cxn += fmt.Sprintf("UID=%s; PWD=%s; ", c.User, c.Password)
	}

	// Database
	if c.Database != "" {
		cxn += fmt.Sprintf("Database=%s; ", c.Database)
	}

	// App Name
	if c.AppName != "" {
		cxn += fmt.Sprintf("App=%s; ", c.AppName)
	}

	// MultisubnetFailover
	if c.MultiSubnetFailover {
		cxn += "MultiSubnetFailover=Yes; "
	}

	return cxn, nil
}

// Parse converts a connection string into a Connection
func Parse(s string) (Connection, error) {
	var c Connection
	var err error

	// Remove any spaces at the beginning and end
	s = strings.TrimSpace(s)

	// remove a trailing ; if present
	// split the string on the ;
	attribs := strings.Split(strings.TrimSuffix(s, ";"), ";")

	for _, a := range attribs {

		// get the bits around the = sign
		p := strings.Split(a, "=")

		// all are in format: attribute ::= attribute-keyword=[{]attribute-value[}]
		if len(p) != 2 {
			return c, errors.New("bad attrib: " + a)
		}

		k := strings.ToLower(strings.TrimSpace(p[0]))
		v := strings.TrimSpace(p[1])

		// remove the possible { and } around a value
		v = strings.TrimPrefix(strings.TrimSuffix(v, "}"), "{")

		switch k {
		case "driver":
			err = c.SetDriver(v)
			if err != nil {
				return c, errors.Wrap(err, "setdriver")
			}

		case "server", "address", "addr":
			c.Server = v

		case "uid", "user id":
			c.User = v

		case "pwd", "password":
			c.Password = v

		case "database":
			c.Database = v

		case "app", "application name":
			c.AppName = v

		case "trusted_connection":
			if strings.ToLower(v) == "yes" {
				c.Trusted = true
			}

		case "multisubnetfailover":
			if strings.ToLower(v) == "yes" {
				c.MultiSubnetFailover = true
			}
		}
	}

	return c, err
}
