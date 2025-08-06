package xe

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"golang.org/x/text/unicode/norm"

	"github.com/billgraziano/xelogstash/pkg/logstash"
	"github.com/pkg/errors"
)

var errorMessageErrorRegex = regexp.MustCompile(`Error:\s(?P<error_number>\d+),\sSeverity:\s(?P<severity>\d+),\sState:\s(?P<state>\d+)`)
var spaceRegex = regexp.MustCompile(`\s+`)
var clientAddressRegex = regexp.MustCompile(`\[CLIENT: (?P<xe_client_address>[^][]*)]`)
var processLogon = "logon"

// Event is a key value of entries for the XE event
type Event map[string]interface{}

type xmlDataType struct {
	Name string `xml:"name,attr"`
}

type xmlData struct {
	Name  string      `xml:"name,attr"`
	Value string      `xml:"value"`
	Text  string      `xml:"text"`
	Type  xmlDataType `xml:"type"`
}

type xmlAction struct {
	Name  string      `xml:"name,attr"`
	Value string      `xml:"value"`
	Text  string      `xml:"text"`
	Type  xmlDataType `xml:"type"`
}

// XMLEventData holds the root for XE XML data
type XMLEventData struct {
	Name         string      `xml:"name,attr"`
	TimeStamp    time.Time   `xml:"timestamp,attr"`
	DataValues   []xmlData   `xml:"data"`
	ActionValues []xmlAction `xml:"action"`
}

// Name returns the "name" attribute from the event
// It returns an empty string if not found or not a string
func (e *Event) Name() string {
	i, ok := (*e)["name"]
	if !ok {
		return ""
	}
	s, ok := i.(string)
	if !ok {
		return ""
	}
	return s
}

// Timestamp returns the "timestamp" attribute from the event
// It returns the zero value if it doesn't exist
func (e Event) Timestamp() time.Time {
	zero := time.Time{}
	i, ok := e["timestamp"]
	if !ok {
		return zero
	}
	ts, ok := i.(time.Time)
	if !ok {
		return zero
	}
	return ts
}

// GetInt64 returns an integer value.  The raw map value must be an int64.
func (e *Event) GetInt64(key string) (int64, bool) {
	raw, ok := (*e)[key]
	if !ok {
		return 0, false
	}

	i64, ok := raw.(int64)
	if !ok {
		return 0, false
	}

	return i64, true
}

// GetIntFromString returns an int64 from a string'd interface
func (e *Event) GetIntFromString(key string) (int64, bool) {
	v, exists := (*e)[key]
	if !exists {
		return 0, false
	}
	str := fmt.Sprintf("%v", v)
	i64, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, false
	}
	return i64, true
}

// CacheSize returns the number of items in the cache
func (i *SQLInfo) CacheSize() int {
	return len(i.Actions) + len(i.Fields)
}

func (i *SQLInfo) getDataValue(object string, x xmlData, eventData string) interface{} {
	if x.Text != "" {
		return x.Text
	}
	dtkey := FieldTypeKey{Object: object, Name: x.Name}
	dt, found := i.Fields[dtkey]
	if !found {
		return x.Value
	}
	return getValue(x.Name, dt, x.Value, eventData)
}

func getValue(name, datatype, value, eventData string) interface{} {
	var newValue interface{}

	switch datatype {
	case "boolean":
		newValue, _ = strconv.ParseBool(value)
	case "float32":
		newValue, _ = strconv.ParseFloat(value, 32)
	case "float64":
		newValue, _ = strconv.ParseFloat(value, 64)
	case "int8":
		newValue, _ = strconv.ParseInt(value, 10, 8)
	case "int16":
		newValue, _ = strconv.ParseInt(value, 10, 16)
	case "int32":
		newValue, _ = strconv.ParseInt(value, 10, 32)
	case "int64":
		newValue, _ = strconv.ParseInt(value, 10, 64)
	case "uint8":
		newValue, _ = strconv.ParseUint(value, 10, 8)
	case "uint16":
		newValue, _ = strconv.ParseUint(value, 10, 16)
	case "uint32":
		newValue, _ = strconv.ParseUint(value, 10, 32)
	case "uint64":
		newValue, _ = strconv.ParseUint(value, 10, 64)
	case "xml":
		xmlData, err := getInnerXML(eventData, name)
		if err != nil {
			return errors.Wrap(err, "getinnerxml")
		}
		newValue = xmlData
	default:
		newValue = value
	}
	return newValue
}

func (i *SQLInfo) getActionValue(a xmlAction, eventData string) interface{} {
	if a.Text != "" {
		return a.Text
	}
	dt, found := i.Actions[a.Name]
	if !found {
		return a.Value
	}
	var newValue interface{}
	newValue = a.Value
	// hardcode some hacks
	if a.Name == "query_hash" {
		return newValue // leave as string
	}
	newValue = getValue(a.Name, dt, a.Value, eventData)

	return newValue
}

// Parse converts event data into an Event
func Parse(i *SQLInfo, eventData string, beta bool) (Event, error) {

	event := make(Event)

	var ed XMLEventData
	err := xml.Unmarshal([]byte(eventData), &ed)
	if err != nil {
		return event, errors.Wrap(err, "unmarshall")
	}

	for _, d := range ed.DataValues {
		dataValue := i.getDataValue(ed.Name, d, eventData)
		// some events have their own timestamp field
		// we use the event name underscore timestamp in this case
		// memory_broker_ring_buffer_recorded_timestamp
		if d.Name == "timestamp" {
			key := ed.Name + "_" + d.Name
			event[key] = dataValue
		} else {
			event[d.Name] = dataValue
		}
	}

	for _, a := range ed.ActionValues {
		actionValue := i.getActionValue(a, eventData)
		event[a.Name] = actionValue
	}

	event["name"] = ed.Name
	event["timestamp"] = ed.TimeStamp

	if ed.Name == "xml_deadlock_report" {
		xmldeadlockreport, err := getInnerXML(eventData, "xml_report")
		if err != nil {
			return event, errors.Wrap(err, "getinnerxml")
		}
		event["xml_deadlock_report"] = xmldeadlockreport
	}

	if ed.Name == "blocked_process_report" {
		blockedProcessReport, err := getInnerXML(eventData, "blocked_process")
		if err != nil {
			return event, errors.Wrap(err, "getinnerxml")
		}
		event["blocked_process_report"] = blockedProcessReport
	}

	if ed.Name == "errorlog_written" {
		event.parseErrorLogMessage()
	}

	// Fixup 2008 used "error" but 2012+ used error_number remap this
	event.Rename("error", "error_number")
	severity := event.getSeverity()
	event["xe_severity_value"] = severity
	event["xe_severity_keyword"] = severity.String()

	if i.Domain != "" {
		event.Set("mssql_domain", i.Domain)
	}
	event.Set("mssql_computer", i.Computer)
	event.Set("mssql_server_name", i.Server)
	event.Set("mssql_version", i.Version)
	event.Set("mssql_product_version", i.ProductVersion)

	if len(i.AvailibilityGroups) > 0 {
		event.Set("mssql_ag", i.AvailibilityGroups)
	}
	if len(i.Listeners) > 0 {
		event.Set("mssql_ag_listener", i.Listeners)
	}

	// enrich data
	event.SetIfEmpty("server_instance_name", i.Server)
	event.setDatabaseName(i)

	event.SetAppSource()
	// set calc_description
	desc := event.getDescription()
	if len(desc) > 0 {
		event["xe_description"] = desc
	}

	category := event.getCategory()
	if len(category) > 0 {
		event["xe_category"] = category
	}

	if ed.Name == "error_reported" {
		event.parseErrorReported(i, desc)
	}

	if beta {
		// writes_mb, cpu_time_sec, etc.
		event.SetExtraUnits()
	}

	return event, nil
}

func (e *Event) parseErrorReported(i *SQLInfo, desc string) {
	// get the error number and see if it is a login error
	errnum, ok := e.GetInt64("error_number")
	if ok {
		_, ok = i.LoginErrors[errnum]
		if ok {
			e.Set("login_failed", desc)
		}
	}

	// see if we have a [CLIENT: 10.10.128.85] phrase in the message
	rawMsg := e.GetString("message")
	clientMatch := clientAddressRegex.FindStringSubmatch(rawMsg)
	if clientMatch != nil {
		// name is set in the regex
		for i, name := range clientAddressRegex.SubexpNames() {
			if i != 0 && name != "" {
				val := clientMatch[i]
				val = left(val, 100, "...") // just in case
				e.Set(name, val)
			}
		}
	}
}

func (e *Event) parseErrorLogMessage() {
	rawMsg := e.GetString("message")
	e.Set("errorlog_raw", left(rawMsg, 8000, "..."))

	// I'm not really sure I want to do this
	// I think it will replace multiple spaces inside quotes
	// but it seems to work so far
	rawMsg = spaceRegex.ReplaceAllString(rawMsg, " ")
	if rawMsg == "" || rawMsg == " " { // we have no message
		return
	}

	// See if this is an error with an error number
	// If so, we will set the error fields
	foundError := false
	errorMatch := errorMessageErrorRegex.FindStringSubmatch(rawMsg)
	if errorMatch != nil {
		for i, name := range errorMessageErrorRegex.SubexpNames() {
			if i != 0 && name != "" {
				val, err := strconv.Atoi(errorMatch[i])
				// if it isn't a nubmer, just skip it
				if err != nil {
					continue
				}
				foundError = true
				e.Set(name, int64(val))
			}
		}
	}

	// see if we have a [CLIENT: 10.10.128.85] phrase in the message
	clientMatch := clientAddressRegex.FindStringSubmatch(rawMsg)
	if clientMatch != nil {
		for i, name := range clientAddressRegex.SubexpNames() {
			if i != 0 && name != "" {
				val := clientMatch[i]
				val = left(val, 100, "...") // just in case
				e.Set(name, val)
			}
		}
	}

	// Start trying to figure out just the message part
	ff := strings.Split(rawMsg, " ")
	if len(ff) < 4 {
		e.Set("errorlog_message", left(rawMsg, 8000, "..."))
		return
	}
	timestampAndProcess := strings.Join(ff[0:3], " ")

	process := strings.ToLower(strings.TrimSpace(ff[2]))
	e.Set("errorlog_process", left(process, 100, "..."))
	msg := strings.TrimSpace(strings.Join(ff[3:], " "))

	// Logon repeats the timestamp and process
	// so we will remove it
	if process == processLogon {
		msg = strings.Replace(msg, timestampAndProcess, "", -1)
	}
	e.Set("errorlog_message", left(msg, 8000, "..."))

	if foundError && process == processLogon {
		e.Set("login_failed", left(msg, 8000, "..."))
	}
}

// setDatabaseName sets the name if we have a database_id
// This is mainly used for AG health events
func (e *Event) setDatabaseName(i *SQLInfo) {
	// We need a database_id, but not a database_name
	_, exists := (*e)["database_name"]
	if exists {
		return
	}
	dbid, exists := (*e).getDatabaseID()
	if !exists {
		return
	}
	dbv, exists := i.Databases[dbid]
	if !exists {
		return
	}
	ts := (*e).GetTime("timestamp")
	if ts.IsZero() {
		return
	}

	// if the event timestamp is before database.create_date, we are done
	if ts.Before(dbv.CreateDate) {
		return
	}
	(*e)["database_name"] = dbv.Name
}

func (e *Event) getSeverity() logstash.Severity {
	name := e.Name()
	if name == "error_reported" {
		var severity int
		severity, ok := (*e)["severity"].(int)
		if !ok {
			return logstash.Error
		}
		if severity >= 11 {
			return logstash.Error
		}
	}

	if name == "xml_deadlock_report" || name == "lock_deadlock_chain" {
		return logstash.Error
	}

	if name == "wait_info_external" ||
		name == "wait_info" ||
		name == "scheduler_monitor_non_yielding_ring_buffer_recorded" ||
		name == "blocked_process_report" {
		return logstash.Warning
	}

	if name == "sp_server_diagnostics_component_result" {
		switch state := e.GetString("state"); state {
		case "WARNING":
			return logstash.Warning
		case "ERROR":
			return logstash.Error
		default:
			return logstash.Info
		}
	}

	return logstash.Info
}

// getCategroy assigns a category based on the event.
// It groups TSQL, HADR, deadlocks, etc. together.
func (e *Event) getCategory() string {
	name := e.Name()
	switch name {
	case "sql_batch_completed", "rpc_completed", "sp_statement_completed", "sql_statement_completed":
		return "tsql"
	case "lock_deadlock_chain", "xml_deadlock_report":
		return "deadlock"
	case "hadr_db_partner_set_sync_state", "alwayson_ddl_executed", "availability_replica_manager_state_change", "availability_replica_state":
		return "hadr"
	case "agent_job", "agent_job_step":
		return "agent"
	case "wait_info", "wait_info_external":
		return "wait"
	default:
		return name
	}
}

// getDescription return a short human readable description of the event
func (e *Event) getDescription() string {

	/*
		Need to decode xe map values at some point
		select *
		from sys.dm_xe_map_values
		where [name] = 'wait_types'
		and map_key in ( 66, 427)

	*/
	name := e.Name()
	switch name {
	case "attention":
		return e.getSQLDescription("sql_text")
	case "login":
		var msg string
		acct := e.GetString("server_principal_name")
		if len(acct) > 0 {
			msg = acct + " "
		}

		client := e.GetString("client_hostname")
		if len(client) > 0 {
			msg += "from " + client + " "
		}

		app := e.GetString("client_app_name")
		if len(app) > 0 {
			msg += "using " + app
		}
		msg = strings.TrimSpace(msg)

		return msg

	case "errorlog_written":
		str := e.GetString("errorlog_message")
		if str != "" {
			return str
		}
		return e.GetString("message")

	case "sql_batch_completed":
		return e.getSQLDescription("sql_text", "batch_text")

	case "rpc_completed":
		return e.getSQLDescription("statement", "sql_text")

	case "sp_statement_completed":
		return e.getSQLDescription("statement", "sql_text")

	case "sql_statement_completed":
		return e.getSQLDescription("statement", "sql_text")

	case "error_reported":
		var msg string
		var p []string

		errnum := e.GetString("error_number")
		if len(errnum) > 0 {
			p = append(p, fmt.Sprintf("Msg %s", errnum))
		}

		severity := e.GetString("severity")
		if len(severity) > 0 {
			p = append(p, fmt.Sprintf("Level %s", severity))
		}

		state := e.GetString("state")
		if len(state) > 0 {
			p = append(p, fmt.Sprintf("State %s", state))
		}

		if len(p) > 0 {
			msg = fmt.Sprintf("(%s) ", strings.Join(p, ", "))
		}
		msg += e.GetString("message")

		return msg
	case "wait_info", "wait_info_external":
		var s, dur string
		var t int64
		duration, exists := (*e)["duration"]
		if exists {
			t, _ = strconv.ParseInt(fmt.Sprintf("%d", duration), 10, 64)
			dur = roundDuration(time.Duration(t) * time.Millisecond)
		}
		wt := e.GetString("wait_type")
		sqltext := e.GetString("sql_text")
		if wt != "" {
			if t > 0 {
				s = fmt.Sprintf("(%s) %s", dur, wt)
			} else {
				s = wt
			}
		}
		if len(sqltext) > 200 {
			sqltext = sqltext[:199] + "..."
		}

		if sqltext != "" {
			s += fmt.Sprintf(" (%s)", sqltext)
		}
		return s

	case "object_altered":
		return fmt.Sprintf("ALTER %s..%s (%s)", e.GetString("database_name"), e.GetString("object_name"), e.GetString("object_type"))
	case "object_created":
		return fmt.Sprintf("CREATE %s..%s (%s)", e.GetString("database_name"), e.GetString("object_name"), e.GetString("object_type"))
	case "object_deleted":
		return fmt.Sprintf("DELETE %s..%s (%s)", e.GetString("database_name"), e.GetString("object_name"), e.GetString("object_type"))
	case "lock_deadlock_chain":
		return e.GetString("resource_description")
	case "xml_deadlock_report":
		return "xml_deadlock_report"
	case "hadr_db_partner_set_sync_state":
		return fmt.Sprintf("%s: %s -> %s (%s)", e.GetString("database_name"), e.GetString("commit_policy"), e.GetString("commit_policy_target"), e.GetString("sync_state"))
	case "hadr_trace_message":
		return e.GetString("hadr_message")
	case "blocked_process_report":
		var s string
		r := e.GetResourceUsageDesc()
		if r != "" {
			s = fmt.Sprintf("(%s) ", r)
		}
		s += fmt.Sprintf("%s: (%s-%s[%s])", e.GetString("database_name"), e.GetString("resource_owner_type"), e.GetString("lock_mode"), e.GetString("object_id"))
		return s
	case "alwayson_ddl_executed":
		return fmt.Sprintf("(%s) %s", e.GetString("ddl_phase"), e.GetString("statement"))
	case "availability_replica_manager_state_change":
		return fmt.Sprintf("state: %s", e.GetString("current_state"))
	case "availability_replica_state_change":
		return fmt.Sprintf("%s: %s -> %s", e.GetString("availability_group_name"), e.GetString("previous_state"), e.GetString("current_state"))
	case "availability_replica_state":
		return fmt.Sprintf("%s: %s", e.GetString("availability_group_name"), e.GetString("current_state"))
	case "database_mirroring_state_change":
		return fmt.Sprintf("%s: %s", e.GetString("database_name"), e.GetString("state_change_desc"))
	case "sql_exit_invoked":
		return e.GetString("shutdown_option")
	case "sp_server_diagnostics_component_result":
		return fmt.Sprintf("(%s:%s) %s", e.GetString("component"), e.GetString("state"), e.GetString("data"))
	case "database_file_size_change":
		var str string
		dbname := e.GetString("database_name")
		if len(dbname) > 0 {
			str += fmt.Sprintf("%s: ", dbname)
		}
		fileName := e.GetString("file_name")
		if len(fileName) > 0 {
			str += fmt.Sprintf("%s: ", fileName)
		}
		change_kb, ok := e.GetIntFromString("size_change_kb")
		if ok {
			change, units := kbtombstring(change_kb)
			str += fmt.Sprintf("%d %s", change, units)
		}
		durtn, ok := e.GetIntFromString("duration")
		if ok {
			str += fmt.Sprintf(" (%dms)", durtn/1000)
		}
		return str
	}

	return ""
}

// kbtombstring accepts KB (1024 type) and returns either KB or MB
// if it is an even multiple of 1024
func kbtombstring(kb int64) (int64, string) {
	units := "KB"
	change := kb
	if change >= 1024 && change%1024 == 0 {
		units = "MB"
		change = change / 1024
	}
	return change, units
}

func (e *Event) getSQLDescription(name ...string) string {
	var txt string
	for _, fld := range name {
		txt = e.GetString(fld)
		if len(txt) > 0 {
			break
		}
	}

	var s string
	r := e.GetResourceUsageDesc()
	if len(r) > 0 {
		s = fmt.Sprintf("(%s) ", r)
	}

	s += left(txt, 300, "...")
	return s
}

// GetString returns a value as a string
func (e *Event) GetString(key string) string {
	v, exists := (*e)[key]
	if !exists {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// getDatabaseID returns an int64 value
func (e *Event) getDatabaseID() (int64, bool) {
	var i int64
	v, exists := (*e)["database_id"]
	if !exists {
		return 0, false
	}
	s := fmt.Sprintf("%v", v)
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, false
	}

	return i, true
}

// GetTime returns a time.Time if the value is time.Time
// Otherwise is returns a zero value time
func (e *Event) GetTime(key string) time.Time {
	var z time.Time
	v, exists := (*e)[key]
	if !exists {
		return z
	}
	t, ok := v.(time.Time)
	if !ok {
		return z
	}
	return t
}

// GetResourceUsageDesc returns a compressed CPU, Reads, Writes, Duration field
func (e *Event) GetResourceUsageDesc() string {
	var usage []string
	cpu, exists := (*e)["cpu_time"]
	if exists {
		t, _ := strconv.ParseInt(fmt.Sprintf("%v", cpu), 10, 64)
		usage = append(usage, fmt.Sprintf("CPU: %s", roundDuration(time.Duration(t)*time.Microsecond)))
	}

	lr, exists := (*e)["logical_reads"]
	if exists {
		t, _ := strconv.ParseInt(fmt.Sprintf("%v", lr), 10, 64)
		t = t * 8192 // to bytes
		if t > 0 {
			v := uint64(t)
			usage = append(usage, fmt.Sprintf("L: %s", humanize.Bytes(v)))
		}
	}

	pr, exists := (*e)["physical_reads"]
	if exists {
		t, _ := strconv.ParseInt(fmt.Sprintf("%v", pr), 10, 64)
		t = t * 8192 // to bytes
		if t > 0 {
			v := uint64(t)
			usage = append(usage, fmt.Sprintf("P: %s", humanize.Bytes(v)))
		}
	}

	w, exists := (*e)["writes"]
	if exists {
		t, _ := strconv.ParseInt(fmt.Sprintf("%v", w), 10, 64)
		t = t * 8192 // to bytes
		if t > 0 {
			v := uint64(t)
			usage = append(usage, fmt.Sprintf("W: %s", humanize.Bytes(v)))
		}
	}

	duration, exists := (*e)["duration"]
	if exists {
		t, _ := strconv.ParseInt(fmt.Sprintf("%v", duration), 10, 64)
		usage = append(usage, fmt.Sprintf("D: %s", roundDuration(time.Duration(t)*time.Microsecond)))
	}

	// convert to string, the back to int64, then math, then string
	return strings.Join(usage, "; ")
}

// getInnerXML gets the XML bit of a deadlock report from data=xml_report
func getInnerXML(eventData string, dataName string) (string, error) {

	type xmlDataNested struct {
		Name  string `xml:"name,attr"`
		Value string `xml:",innerxml"`
		Text  string `xml:"text"`
	}

	//XMLEventData holds the root for XE XML data
	type XMLWrapper struct {
		Name       string          `xml:"name,attr"`
		TimeStamp  time.Time       `xml:"timestamp,attr"`
		DataValues []xmlDataNested `xml:"data"`
	}

	var wrapper XMLWrapper
	err := xml.Unmarshal([]byte(eventData), &wrapper)
	if err != nil {
		return "", errors.Wrap(err, "unmarshallInner")
	}

	for _, v := range wrapper.DataValues {
		if v.Name == dataName {
			return v.Value, nil
		}
	}

	return "", nil
}

// ToJSON marshalls to a byte array
func (e *Event) ToJSON() (string, error) {
	jsonBytes, err := json.Marshal(e)
	if err != nil {
		return "", errors.Wrap(err, "marshall")
	}

	jsonString := string(jsonBytes)

	return jsonString, nil
}

// Set assigns a string value to a key in the event
func (e *Event) Set(key string, value interface{}) {
	(*e)[key] = value
}

// SetIfEmpty assigns a value at the key if it doesn't already exist
func (e *Event) SetIfEmpty(key string, value interface{}) {
	_, exists := (*e)[key]
	if !exists {
		e.Set(key, value)
	}
}

// Copy value from srckey to newkey
func (e *Event) Copy(srckey, newkey string) {
	// does srckey exist?
	_, ok := (*e)[srckey]
	if !ok {
		return
	}
	v := (*e)[srckey]
	(*e)[newkey] = v
}

// Rename old key to new key
func (e *Event) Rename(oldkey, newkey string) {
	// does old key exist?
	_, ok := (*e)[oldkey]
	if !ok {
		return
	}
	(*e).Copy(oldkey, newkey)
	delete((*e), oldkey)
}

// SetAppSource creates the xecap_login_app_client event which is
// server_principal_name - client_app_name (client_hostname)
// and xecap_login_app
func (e *Event) SetAppSource() {
	var s string
	login, ok := (*e)["server_principal_name"]
	if ok {
		s = fmt.Sprintf("%v", login)
	}

	app, ok := (*e)["client_app_name"]
	if ok {
		appstring := fmt.Sprintf("%v", app)
		if len(s) > 0 && len(appstring) > 0 {
			s = s + " - "
		}
		s = s + appstring
	}
	if len(s) > 0 {
		(*e)["xe_acct_app"] = s
	}

	client, ok := (*e)["client_hostname"]
	if ok {
		cs := fmt.Sprintf("%v", client)
		if len(cs) > 0 {
			if len(s) > 0 {
				s = s + " (" + cs + ")"
			} else {
				s = cs
			}
		}
	}
	if len(s) > 0 {
		(*e)["xe_acct_app_client"] = s
	}
}

// left returns the left most characters of a string
// it should handle unicode/utf8 without splitting characters
// It is designed for big strings of SQL statements
// it uses a sketchy test to see if the string is already short enough
// if the string is trimmed, suffix is added at the end
// https://stackoverflow.com/questions/61353016/why-doesnt-golang-have-substring
// This seems better: https://stackoverflow.com/questions/46415894/golang-truncate-strings-with-special-characters-without-corrupting-data
func left(str string, n int, suffix string) string {
	if n == 0 {
		return ""
	}
	if n < 1 { // -1 is the whole string
		return str
	}
	if len(str) <= n { // fewer bytes than we want characters
		return str
	}
	str = norm.NFC.String(str)
	result := str
	chars := 0
	trimmed := false
	// https://go.dev/doc/effective_go#for
	// for over a string ranges over the runes.
	// i is bumped to the first byte to each rune,
	// then we just count runes/characters
	for i := range str {
		if chars >= n {
			result = str[:i]
			trimmed = true
			break
		}
		chars++
	}
	if trimmed {
		result += suffix
	}
	return result
}

func roundDuration(d time.Duration) string {

	var s string
	// 17h3m
	if d.Minutes() > 90 {
		h := d.Truncate(time.Hour)
		//fmt.Println(d.Hours(), h.Hours())
		m1 := d.Nanoseconds() - h.Nanoseconds()
		m2 := time.Duration(m1) * time.Nanosecond
		m := m2.Truncate(time.Minute)
		//fmt.Println(m2, m)
		s = fmt.Sprintf("%.0fh", h.Hours())
		if m.Minutes() != 0.0 {
			s += fmt.Sprintf("%.0fm", m.Minutes())
		}
		return s
	}
	// 12m3s
	if d.Seconds() > 99 {
		m := d.Truncate(time.Minute)
		s := (time.Duration(d.Nanoseconds()-m.Nanoseconds()) * time.Nanosecond).Truncate(time.Second)
		//fmt.Println(s)
		if s.Seconds() != 0 {
			return fmt.Sprintf("%.0fm%.0fs", m.Minutes(), s.Seconds())
		}
		return fmt.Sprintf("%.0fm", m.Minutes())
	}

	if d.Seconds() >= 1 {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}

	if d.Nanoseconds() > (1 * time.Millisecond).Nanoseconds() {
		return fmt.Sprintf("%dms", d.Round(time.Millisecond).Nanoseconds()/1000000)
	}

	if d.Nanoseconds() > (1 * time.Millisecond).Nanoseconds() {
		return fmt.Sprintf("%dms", d.Round(time.Millisecond).Nanoseconds()/1000000)
	}

	if d.Nanoseconds() > (1 * time.Microsecond).Nanoseconds() {
		return fmt.Sprintf("%dμs", d.Round(time.Microsecond).Nanoseconds()/1000)
	}

	return fmt.Sprintf("%dns", d.Nanoseconds())
}
