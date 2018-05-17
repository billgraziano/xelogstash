# Fixes

  
  - Because config.Copies is map[string]string, I can't put the same source in twice.
      So this needs to be an array or some such.  Ugh.
  - Logging: Need a row saying where I am sending events
      - Need a row saying where I am sending application logs
  - if no timestamp field is specified, default to "timestamp".  How do I over write this?
  - Check for permissions to query msdb.sys.sysjobhistory & put up a warning if 
    we don't have it
  - Maybe don't log events with zero rows.  Or log the debug level.

# Some metrics

* 350,000 login events yields a 23MB status.0 file
* It processed 3,500 rows per second

# Fields

Prefix: xecap - fields I make up to enrich

* xecap_login_app_client
* xecap_login_app
* TODO: xecap_severity_value -- numeric syslog values (0,1,2,3,4,5,6,7)
* TODO: xecap_severity_keyword -- "emerg", "alert", "crit", "err", "warning", "notice", "info", "debug"
* TODO: xecap_description -- try to build a good text description of the event
  * error_reported includes severity and state
  * SQL includes CPU, Reads, Writes, Duration, plus first 200 characters of the text

Prefix: xe - fields from the XE 

* xe_session_name
* xe_file_name
* xe_file_offset

Prefix: mssql - fields I query from SQL Server

* mssql_fqdn - FQDN as reported by SQL Server
* mssql_computer - ServerProperty('MachineName')
* mssql_domain - fully qualified domain (@domain.com)
* mssql_server_name -- @@SERVERNAME 
* mssql_version - SQL Server 2016 SP1 (version + product level)

 for the value in adds, put a number, bool, or float in '' to make it a string

# Design Notes

* logstash.Record has Timestamp and field KV map[string]interface{}
* Every entry that goes into the map is prefixed with json_payload_field and a dot.
* Settings has json_timestamp_field & json_payload_field.
  * If json_payload_field is empty, then everything will be at the root level
  * if json_timestamp_field is empty then use "timestamp"
* The adds, merges, moves just go againts the map using fully-qualifed dot notation
  * All the sets and adds should lower-case the key
* Right before I marshall, add in the timestamp using json_field_timestamp
  * or just "timestamp" if no field
* Marshall using the sjson so I get nested values
* Move logstash setting to the apps section

## JSON Mods

* Include some fancy variables the injected values: 
  * $(EXENAME) is the fully qualified EXE name
  * $(EXEVERSION)
  * $(EXECPROCID)
  * $(EXENAMEONLY)
  * $(HOST)
  * Set these from the GO os package
  * These can be used in the ADD events
* Add with a blank value should delete a key
* Add with '' should set to empty string

# Issues 

  * What should happen when a session isn't running?  Warning for now.
    * May an error on stopped session.  Otherwise just a warning.
    * error_on_bad_session: true; false prints a warning

# Severity

* INFO, WARN, ERR
* INFO - Every standard message that doesn't have another setting
* WARN
  * Right now, only a stopped logstash session or one we can't query
  * Maybe those stupid 17830 errors about reusing SPIDs
* ERR - Any non-standard event or thing I know is an error
* Create an `xecap_severity` numeric scale matching syslog
  * INFO - 6
  * WARN - 5
  * ERR - 3
* Each SQL event is assigned one of these severities
* Include some way to map a particular severity to some type of JSON field and value
  * Eventually, have three config settings: info, warn, err.  Each is a K:V.  For that type
    of event, inject that KV into the event.

# Database

* Use a local Badger installation instead of text files
* -export parameter that extracts the database to a JSON file 
  * -backup does the same but doesn't process any events
* -import parameter that replaces the database with a JSON file
  * -restore does the same but doesn't process any events

# Features

* If a session is stopped or not available, just issue a warning.
  - xecap_description on errors
    (Msg 8134, Level 16, State 1, Line 1) and then the error text

- For system_health and AG health, they don't capture the server name,
      So I need to set server_instance_name

* Mail logs
* Job success and failure
 

excludedXE = [
    "connectivity_ring_buffer_recorded",
    "memory_broker_ring_buffer_recorded",
    "sp_server_diagnostics_component_result",
    "scheduler_monitor_system_health_ring_buffer_recorded",
    "security_error_ring_buffer_recorded"
    ]

| event                   | cpu             | duration |
|-------                  |-----            |----------|
| rpc_completed           | microseconds (1)| microseconds
| sp_statement_completed  | microseconds    | microseconds
| sql_batch_completed     | microseconds (2)| microseconds
| sql_statement_completed | microseconds    | microseconds 

(1) sys.dm_xe_object_columns description is null

(2) sys.dm_xe_object_columns says milliseconds but that seems wrong

__And this is ALL messed up in SQL Server 2008.__