package app

import (
	"database/sql"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/davidscholberg/go-durationfmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func safeClose(c io.Closer, err *error) {
	cerr := c.Close()
	if cerr != nil {
		log.Error("safeClose: ", cerr)
		if *err == nil {
			*err = cerr
		}
	}
}

// SQLInfo stores details about the server we connected to
type SQLInfo struct {
	Server   string
	Domain   string
	Computer string
	// FQDN           string
	ProductLevel   string
	ProductRelease string
	Version        string
	ProductVersion string // ServerProperty('ProductVersion') in #.#.#.#
}

// GetInstance returns the instance and domain name
func GetInstance(db *sql.DB, session, serverOverride, domainOverride string) (info SQLInfo, err error) {

	query := `
	SET NOCOUNT ON;
	SELECT	@@SERVERNAME AS [ServerName]
		,COALESCE(DEFAULT_DOMAIN(), '') AS [DomainName]
		,CAST(SERVERPROPERTY('MachineName') as nvarchar(128)) AS [Computer]
		,CAST(COALESCE(SERVERPROPERTY('ProductLevel'), '') as nvarchar(128)) AS ProductLevel
		,COALESCE(CAST(SERVERPROPERTY('ProductMajorVersion') as NVARCHAR(128))  + '.' + CAST(SERVERPROPERTY('ProductMinorVersion') as NVARCHAR(128)),'') AS ProductRelease
		,COALESCE(CAST(SERVERPROPERTY('ProductVersion') AS NVARCHAR(128)), '') as [ProductVersion];
	`
	row := db.QueryRow(query)
	err = row.Scan(&info.Server, &info.Domain, &info.Computer, &info.ProductLevel, &info.ProductRelease, &info.ProductVersion)
	if err != nil {
		return info, errors.Wrap(err, "scan")
	}

	if serverOverride != "" {
		info.Server = serverOverride
		info.Computer = serverOverride
	}
	if domainOverride != "" {
		info.Domain = domainOverride
	}

	var v string
	switch info.ProductRelease {
	case "16.0":
		v = "SQL Server 2022"
	case "15.0":
		v = "SQL Server 2019"
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
		v = fmt.Sprintf("SQL Server %s", info.ProductRelease)
	}
	info.Version = fmt.Sprintf("%s %s", v, info.ProductLevel)

	return info, nil
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

// fmtduration a duration to 5d3h, 2h4m, 2m1s
func fmtduration(d time.Duration) string {
	if d.Hours() >= 24 {
		str, err := durationfmt.Format(d, "%dd%hh")
		if err != nil {
			return d.String()
		}
		return str
	}
	if d.Hours() >= 1 {
		str, err := durationfmt.Format(d, "%hh%mm")
		if err != nil {
			return d.String()
		}
		return str
	}
	str, err := durationfmt.Format(d, "%mm%ss")
	if err != nil {
		return d.String()
	}
	return str
}
