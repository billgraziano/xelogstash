package xe

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/billgraziano/xelogstash/pkg/dbx"
	"github.com/pkg/errors"
)

// SQLInfo stores cached info about the server we connected to
type SQLInfo struct {
	Server   string
	Domain   string
	Computer string

	AvailibilityGroups []string
	Listeners          []string

	ProductLevel   string // ProductLevel holds "SP1"
	ProductRelease string // ProductRelease holds "13.0"
	Version        string // Version holds "SQL Server 2013"
	ProductVersion string // ProductVersion holds "13.0.5101.9"

	Fields       map[FieldTypeKey]string
	Actions      map[string]string
	MapValues    map[MapValueKey]string
	Databases    map[int64]*Database
	LoginErrors  map[int64]bool
	LoggedErrors Set[int64]

	DB *sql.DB
}

// Database holds some basic information about a database on the server
type Database struct {
	Name       string
	CreateDate time.Time
}

// FieldTypeKey is the key for fields structure
type FieldTypeKey struct {
	Object string
	Name   string
}

// MapValueKey is the lookup to the XE map value
type MapValueKey struct {
	Name   string
	MapKey int
}

// NewSQLInfo gets basic SQL Server info and lookup values
func NewSQLInfo(driver, cxnstring, serverOverride, domainOverride string) (info SQLInfo, err error) {
	//func GetSQLInfo(fqdn string, user, password string) (info SQLInfo, err error) {
	info.Fields = make(map[FieldTypeKey]string)
	info.Actions = make(map[string]string)
	info.MapValues = make(map[MapValueKey]string)
	info.AvailibilityGroups = make([]string, 0)
	info.Listeners = make([]string, 0)
	info.LoggedErrors = NewSet[int64]() // roughly 1300 logged errrors as of SQL 2022

	db, err := dbx.Open(driver, cxnstring)
	if err != nil {
		return info, errors.Wrap(err, "opendb")
	}

	err = db.Ping()
	if err != nil {
		if db != nil {
			db.Close()
		}
		return info, errors.Wrap(err, "db.ping")
	}

	info.DB = db

	query := `
	SET NOCOUNT ON;
	SELECT	@@SERVERNAME AS [ServerName]
		,COALESCE(DEFAULT_DOMAIN(), '') AS [DomainName]
		,COALESCE(CAST(SERVERPROPERTY('MachineName') as nvarchar(128)), @@SERVERNAME) AS [Computer]
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
	case "17.0":
		v = "SQL Server 2025"
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

	var object, name, dt string

	// This pattern of rows.open and rows.close may leak objects if we get errors
	// Errors here should be very rare.  We don't have a good way to log them
	// and we already have an error from the query itself.  Leaving it this way
	// for now.

	// Get the action types
	query = "select name, type_name from sys.dm_xe_objects where object_type = 'action' order by type_name;"
	rows, err := db.Query(query)
	if err != nil {
		return info, errors.Wrap(err, "action-query")
	}

	for rows.Next() {
		err = rows.Scan(&name, &dt)
		if err != nil {
			return info, errors.Wrap(err, "action-scan")
		}
		info.Actions[name] = dt
	}
	err = rows.Close()
	if err != nil {
		return info, errors.Wrap(err, "rows.close")
	}

	// Get the fields
	query = `
	select object_name, [name], type_name
	from sys.dm_xe_object_columns
	where column_type = 'data'
	`

	rows, err = db.Query(query)
	if err != nil {
		return info, errors.Wrap(err, "action-query")
	}

	for rows.Next() {
		err = rows.Scan(&object, &name, &dt)
		if err != nil {
			return info, errors.Wrap(err, "action-scan")
		}
		dtkey := FieldTypeKey{Name: name, Object: object}
		info.Fields[dtkey] = dt
	}
	err = rows.Close()
	if err != nil {
		return info, errors.Wrap(err, "rows.close")
	}

	err = info.getMapValues()
	if err != nil {
		return info, errors.Wrap(err, "info.getmapvalues")
	}

	err = info.getDatabases()
	if err != nil {
		return info, errors.Wrap(err, "info.getdatabases")
	}

	err = info.getLoginErrors()
	if err != nil {
		return info, errors.Wrap(err, "info.getloginerrors")
	}

	err = info.getLoggedErrors()
	if err != nil {
		return info, errors.Wrap(err, "info.getloggederrors")
	}

	availGroups, err := stringArrayFromQuery(info.DB, "IF OBJECT_ID('sys.availability_groups') IS NOT NULL SELECT [name] FROM sys.availability_groups ORDER BY [name];")
	if err != nil {
		return info, errors.Wrap(err, "ag")
	}
	info.AvailibilityGroups = availGroups

	listeners, err := stringArrayFromQuery(info.DB, "IF OBJECT_ID('sys.availability_group_listeners') IS NOT NULL select [dns_name] from sys.availability_group_listeners ORDER BY [dns_name];")
	if err != nil {
		return info, errors.Wrap(err, "ag")
	}
	info.Listeners = listeners

	return info, nil
}

func (i *SQLInfo) getMapValues() error {
	i.MapValues = make(map[MapValueKey]string)

	query := "select [name], [map_key], [map_value] from sys.dm_xe_map_values"

	rows, err := i.DB.Query(query)
	if err != nil {
		return errors.Wrap(err, "mapvalue-query")
	}
	var mapKey int
	var name, mapValue string

	for rows.Next() {
		err = rows.Scan(&name, &mapKey, &mapValue)
		if err != nil {
			return errors.Wrap(err, "mapvalue-scan")
		}
		mapValueKey := MapValueKey{Name: name, MapKey: mapKey}
		//info.Fields[dtkey] = dt
		i.MapValues[mapValueKey] = mapValue
	}
	err = rows.Close()
	if err != nil {
		return errors.Wrap(err, "rows.close")
	}
	return nil
}

// getLoginErrors returns all the error messags with "login failed"
func (i *SQLInfo) getLoginErrors() error {
	i.LoginErrors = make(map[int64]bool)

	query := `
		SELECT	message_id 
		FROM 	sys.messages 
		WHERE 	language_id = 1033 
		AND 	[text] LIKE '%login failed%'
		AND		message_id NOT IN (40801);
	`

	rows, err := i.DB.Query(query)
	if err != nil {
		return errors.Wrap(err, "mapvalue-query")
	}
	defer rows.Close()
	var id int64

	for rows.Next() {
		err = rows.Scan(&id)
		if err != nil {
			return errors.Wrap(err, "mapvalue-scan")
		}
		i.LoginErrors[id] = true
	}
	return nil
}

func (i *SQLInfo) getLoggedErrors() error {
	query := `
		SELECT	message_id 
		FROM	sys.messages
		WHERE	is_event_logged = 1 
		AND language_id IN (SELECT msglangid FROM sys.syslanguages WHERE langid = @@LANGID) 
	`
	rows, err := i.DB.Query(query)
	if err != nil {
		return errors.Wrap(err, "mapvalue-query")
	}
	defer rows.Close()
	var id int64

	for rows.Next() {
		err = rows.Scan(&id)
		if err != nil {
			return errors.Wrap(err, "mapvalue-scan")
		}
		i.LoggedErrors.Add(id)
	}
	return nil
}

func (i *SQLInfo) getDatabases() error {
	i.Databases = make(map[int64]*Database)

	query := "SELECT [database_id], [name], [create_date] FROM [sys].[databases];"

	rows, err := i.DB.Query(query)
	if err != nil {
		return errors.Wrap(err, "databases-query")
	}
	var dbid int64

	for rows.Next() {
		var dbv Database
		err = rows.Scan(&dbid, &dbv.Name, &dbv.CreateDate)
		if err != nil {
			return errors.Wrap(err, "databases-scan")
		}

		i.Databases[dbid] = &dbv
	}
	err = rows.Close()
	if err != nil {
		return errors.Wrap(err, "rows.close")
	}
	return nil
}

// stringArrayFromQuery runs a query and returns the rows as a string array
func stringArrayFromQuery(db *sql.DB, query string) ([]string, error) {
	rows, err := db.Query(query)
	if err != nil {
		return []string{}, err
	}
	result := make([]string, 0)
	for rows.Next() {
		var str string
		err = rows.Scan(&str)
		if err != nil {
			return []string{}, err
		}
		result = append(result, str)
	}
	err = rows.Close()
	if err != nil {
		return []string{}, err
	}
	if rows.Err() != nil {
		return []string{}, rows.Err()
	}
	return result, nil
}
