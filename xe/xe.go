package xe

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"

	"github.com/billgraziano/xelogstash/logstash"
	"github.com/pkg/errors"
)

const statementLength = 200

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

//XMLEventData holds the root for XE XML data
type XMLEventData struct {
	Name         string      `xml:"name,attr"`
	TimeStamp    time.Time   `xml:"timestamp,attr"`
	DataValues   []xmlData   `xml:"data"`
	ActionValues []xmlAction `xml:"action"`
}

type dataTypeKey struct {
	Object string
	Name   string
}

// Reader is an object for parsing XE events
type Reader struct {
	actions map[string]string
	fields  map[dataTypeKey]string
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

// XEReader sets the data types of data and actions
// func NewReader(db *sql.DB) (*Reader, error) {
// 	var rdr Reader
// 	var err error

// 	var query string
// 	var object, name, dt string

// 	rdr.fields = make(map[dataTypeKey]string)
// 	rdr.actions = make(map[string]string)

// 	// Get the action types
// 	query = "select name, type_name from sys.dm_xe_objects where object_type = 'action' order by type_name;"
// 	rows, err := db.Query(query)
// 	if err != nil {
// 		return nil, errors.Wrap(err, "action-query")
// 	}

// 	for rows.Next() {
// 		err = rows.Scan(&name, &dt)
// 		if err != nil {
// 			return nil, errors.Wrap(err, "action-scan")
// 		}
// 		rdr.actions[name] = dt
// 	}
// 	rows.Close()

// 	query = `
// 	select object_name, [name], type_name
// 	from sys.dm_xe_object_columns
// 	where column_type = 'data'
// 	`

// 	rows, err = db.Query(query)
// 	if err != nil {
// 		return nil, errors.Wrap(err, "action-query")
// 	}

// 	for rows.Next() {
// 		err = rows.Scan(&object, &name, &dt)
// 		if err != nil {
// 			return nil, errors.Wrap(err, "action-scan")
// 		}
// 		dtkey := dataTypeKey{Name: name, Object: object}
// 		rdr.fields[dtkey] = dt
// 	}
// 	rows.Close()

// 	return &rdr, nil
// }

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
	newValue = value

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
	newValue = getValue(a.Name, dt, a.Value, eventData)

	return newValue
}

// Parse converts event data into an Event
func Parse(i *SQLInfo, eventData string) (Event, error) {

	event := make(Event)

	var ed XMLEventData
	err := xml.Unmarshal([]byte(eventData), &ed)
	if err != nil {
		return event, errors.Wrap(err, "unmarshall")
	}
	event["name"] = ed.Name
	event["timestamp"] = ed.TimeStamp

	for _, d := range ed.DataValues {
		dataValue := i.getDataValue(ed.Name, d, eventData)
		event[d.Name] = dataValue
	}

	for _, a := range ed.ActionValues {
		actionValue := i.getActionValue(a, eventData)
		event[a.Name] = actionValue
	}

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

	// Fixup 2008 used "error" but 2012+ used error_number remap this
	event.Rename("error", "error_number")
	severity := event.getSeverity()
	event["xecap_severity_value"] = severity
	event["xecap_severity_keyword"] = severity.String()

	// set xecap_description
	desc := event.getDescription()
	if len(desc) > 0 {
		event["xecap_description"] = desc
	}

	// if the server_instance_name is empty, then set it
	sn := event.GetString("server_instance_name")
	if sn == "" {
		event.Set("server_instance_name", i.Server)
	}

	// These are stored in ms, so convert to us
	// if ed.Name == "wait_info" || ed.Name == "wait_info_external" {
	// 	durationToMicroSeconds(&event)
	// }

	// // set wait_type_name
	// _, exists := event["wait_type"]
	// if exists {
	// 	err = event.setWaitName(i)
	// 	if err != nil {
	// 		return event, errors.Wrap(err, "event.setwaitname")
	// 	}
	//

	return event, nil
}

// func durationToMicroSeconds(e *Event) {
// 	v, exists := (*e)["duration"]
// 	if !exists {
// 		return
// 	}
// 	d, ok := v.(uint64)
// 	if !ok {
// 		return
// 	}
// 	d = d * 1000
// 	(*e)["duration"] = d
// }

// func (e *Event) setWaitName(i *SQLInfo) error {
// 	// get the wait type as integer
// 	waitType := e.GetString("wait_type")
// 	if waitType == "" {
// 		return nil
// 	}
// 	//var waitTypeInt int
// 	waitTypeInt, err := strconv.ParseInt(waitType, 10, 32)
// 	if err != nil {
// 		return nil
// 	}
// 	if waitTypeInt > math.MaxInt32 || waitTypeInt < math.MinInt32 {
// 		return nil
// 	}
// 	k := MapValueKey{Name: "wait_types", MapKey: int(waitTypeInt)}
// 	name, exists := i.MapValues[k]
// 	if exists {
// 		(*e)["wait_type_name"] = name
// 	}
// 	return nil
// }

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

	if name == "xml_deadlock_report" {
		return logstash.Error
	}

	if name == "wait_info_external" ||
		name == "wait_info" ||
		name == "scheduler_monitor_non_yielding_ring_buffer_recorded" {
		return logstash.Warning
	}

	return logstash.Info
}

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
		return e.GetString("message")

	case "sql_batch_completed":
		return e.getSQLDescription("sql_text")

	case "rpc_completed":
		return e.getSQLDescription("sql_text")

	case "sp_statement_completed":
		return e.getSQLDescription("statement")

	case "sql_statement_completed":
		return e.getSQLDescription("statement")

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
			dur = fmt.Sprintf("%s", roundDuration(time.Duration(t)*time.Millisecond))
		}
		wt := e.GetString("wait_type")
		sqltext := e.GetString("sql_text")
		if wt != "" {
			if t > 0 {
				s = fmt.Sprintf("(%s) %s", dur, wt)
			} else {
				s = fmt.Sprintf("%s", wt)
			}
		}
		if len(sqltext) > 200 {
			sqltext = sqltext[:199] + "..."
		}

		if sqltext != "" {
			s += fmt.Sprintf(" (%s)", sqltext)
		}
		return s
	}

	return ""
}

func (e *Event) getSQLDescription(name string) string {
	var s string
	r := e.GetResourceUsageDesc()
	if len(r) > 0 {
		s = fmt.Sprintf("(%s) ", r)
	}
	txt := e.GetString(name)
	if len(txt) == 0 {
		return ""
	}

	if len(txt) > 200 {
		s += left(txt, 200) + "..."
	} else {
		s += txt
	}
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

// GetResourceUsageDesc returns a compressed CPU, Reads, Writes, Duration field
func (e *Event) GetResourceUsageDesc() string {
	//var s string
	var usage []string
	//var err error
	cpu, exists := (*e)["cpu_time"]
	if exists {
		t, _ := strconv.ParseInt(fmt.Sprintf("%d", cpu), 10, 64)
		usage = append(usage, fmt.Sprintf("CPU: %s", roundDuration(time.Duration(t)*time.Nanosecond)))
	}

	lr, exists := (*e)["logical_reads"]
	if exists {
		t, _ := strconv.ParseInt(fmt.Sprintf("%d", lr), 10, 64)
		if t > 10000 {
			t = t / 1000
			usage = append(usage, fmt.Sprintf("L: %sk", humanize.Comma(t)))
		} else {
			usage = append(usage, fmt.Sprintf("L: %s", humanize.Comma(t)))
		}

	}

	pr, exists := (*e)["physical_reads"]
	if exists {
		t, _ := strconv.ParseInt(fmt.Sprintf("%d", pr), 10, 64)
		if t > 10000 {
			t = t / 1000
			usage = append(usage, fmt.Sprintf("P: %sk", humanize.Comma(t)))
		} else {
			usage = append(usage, fmt.Sprintf("P: %s", humanize.Comma(t)))
		}

	}

	w, exists := (*e)["writes"]
	if exists {
		t, _ := strconv.ParseInt(fmt.Sprintf("%d", w), 10, 64)
		if t > 10000 {
			t = t / 1000
			usage = append(usage, fmt.Sprintf("W: %sk", humanize.Comma(t)))
		} else {
			usage = append(usage, fmt.Sprintf("W: %s", humanize.Comma(t)))
		}

	}

	duration, exists := (*e)["duration"]
	if exists {
		t, _ := strconv.ParseInt(fmt.Sprintf("%d", duration), 10, 64)
		usage = append(usage, fmt.Sprintf("D: %s", roundDuration(time.Duration(t)*time.Nanosecond)))
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

//ToJSON marshalls to a byte array
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
		(*e)["xecap_login_app"] = s
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
		(*e)["xecap_login_app_client"] = s
	}
}

// ToLower sets most fields to lower case.  Fields like message
// and various SQL statements are unchanged
// func (e *Event) ToLower() {
// 	for k, v := range *e {
// 		if k != "message" && k != "timestamp" && k != "sql_text" && k != "statement" && k != "batch_text" {
// 			s, ok := v.(string)
// 			if ok {
// 				(*e)[k] = strings.ToLower(s)
// 			}
// 		}
// 	}
// }

/*

-- some handy queries

SELECT	[name]
FROM	sys.dm_xe_sessions s
JOIN	sys.dm_xe_session_targets t on s.address = t.event_session_address
WHERE	target_name = 'event_file'
ORDER BY s.[name]

;with cte as (
select [name], object_name, type_name
from sys.dm_xe_object_columns
where object_name in ('login')
and column_type = 'data')
select type_name, count(*) from cte group by type_name order by count(*) DESC

select * from sys.dm_xe_object_columns where object_name = 'login'

select * from sys.dm_xe_objects where object_type = 'action' order by [name]

*/

func left(s string, n int) string {
	if n < 0 {
		return s
	}
	if len(s) < n {
		return s
	}
	return s[:n]
}

func roundDuration(d time.Duration) string {
	//h := d.Hours()
	//m := d.Minutes()
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
