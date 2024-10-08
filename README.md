# Extended Event Writer for SQL Server

`sqlxewriter.exe` is an application to pull SQL Server Extended Events and SQL Server Agent job results and write them to various sinks.  It can write to Logstash, Elastic Search, or JSON files.   It runs on the command-line or as a service.    It supports SQL Server 2012 and higher.  It has limited support for SQL Server 2008 (R2). This application replaces `xelogstash.exe`.  *If you were running `xelogstash.exe`, please read this carefully.*


1. [Getting started](#getting-started)
1. [What's New](#whats-new)
2. [Sources and Defaults](#sources)
2. [Controlling JSON](#json)
1. [Adds, Moves, Copies](#adds)
2. [Prefixes and keeping your place](#prefixes)
2. [Application Settings](#app-settings)
3. [Derived Fields](#derived-fields)
3. [Sinks](#sinks)
3. [Beta Features](#beta)
4. [Linux Support](#linux)
4. [Other Notes](#notes)
4. [Building](#building)


## <a name="getting-started"></a>Getting Started
I've found [Visual Studio Code](https://code.visualstudio.com/) to a very good tool to edit TOML files and view log files.  Plus it installs for only the local user by default. _If you use VSCode, copy the `.vscode` directory from the `samples` directory to the root of the project directory._

The application uses a [TOML](https://en.wikipedia.org/wiki/TOML) file for configuration.  Two sample files are included. 

1. Extract the ZIP contents to a directory.  The sample configuration file (`sqlxewriter.toml`) reads the `system_health` session and writes events to a file.
2. If you have a local SQL Server installed, no changes are necessary.
3. Otherwise, edit `sqlxewriter.toml` and change the `fqdn` under ``[[source]]`` to point to a SQL Server.  If you put in a named instance, use double-backslashes instead of a single backslash.  You will need `VIEW SERVER STATE` permission on the instance.
4. From a command-prompt, run "`sqlxewriter.exe -debug`".  (This doesn't send anything to Logstash or Elastic Search yet).  This will pull one set of events and then exit.

This should create an `events` folder and an `xestate` folder.  The `events` folder holds the JSON generated for each event.  The `xestate` folder is used to keep track of the read position in each Extended Event session.  

> NOTE: The permissions on the `xestate` directory are limited. When switching to a service account, be prepared to reset the permissions to grant the service account full control of that directory and the files in it.

### Writing events to a file
The files in the `events` directory are newline-delimited JSON (NDJSON).  The files are rotated every hour.  These files can be written to Elastic Search using [FileBeat](https://www.elastic.co/products/beats/filebeat).  A sample FileBeat configuration file is included.

### Sending to Logstash
To send events to directly Logstash, specify the `logstash` section with a `host` value.  The `host` should be in `host:port` format.  After that you can run the executable with the same parameters. 

````toml
[logstash]
host = "localhost:8888"
````
Now it is writing to both the file and Logstash.

### Command Line Options 
Running `sqlxewriter -?` will display the help for the options.

- `-log` - When running interactively, captures the application output to a log file INSTEAD of standard out.  The log files are located in the `log` subdirectory and named `sqlxewriter_YYYYMMDD.log` based on the start time.  Log files are automatically deleted after 7 days and that's not configurable yet. When running as a service, logs are always written to a file.
- `-debug` - Enables additional debugging output.  If you enable this, it will log each poll of a server.  Otherwise no information is logged on each poll.
- `-loop` - Instead of polling each server once and exiting, it continues to loop and polls each server every minute.  This is only needed when running interactively.  When running as a service, it always loops.
- `-service action` - The two action values are `install` and `uninstall`.  This installs or uninstalls this executable as a service and exits.

### Running as a Windows service
In order to run this as a service in Windows, complete the following steps

1. From an Administrative command prompt, run `sqlxewriter -service install`.  This will install as a service named `sqlxewriter`.  You can find it in the list of services as "XEvent Writer for SQL Server".
1. Configure the service to auto-start
1. Update the service to run as a service account.  This service account should have `VIEW SERVER STATE` permission on each SQL Server it polls.
1. Reset the permissions on the `xestate` directory and ALL files in that directory to grant the service account write permission.
1. When it comes time to update the service, just stop the service and replace the executable.

A similar process should work for Linux.  This uses [github.com/kardianos/service](https://github.com/kardianos/service).  Additional documentation may be found there.

### Running as Linux Service or Container
My experience here is very limited.  Feedback is appreciated.

* This will likely run from the command-line
* Use the `-loop` flag to run continuously
* Consider using `-log` to write logs to a log file
* It should respect CPU quotas after 1.7.9
* The username and password fields for Defaults, Sources, and the Elastic Sink can be set to an environment variable. That looks like this in the configuration file: `password="$(env:VARIABLE_NAME)"`

### Scaling up
1. Changes to the `.toml` require a service restart to take effect unless you set `watch_config = true`.  This includes adding sources.
1. The sample `sqlxewriter.toml` only reads 1 event per server per minute.  This should be set to unlimited (`rows = 0`) or some high number like 20,000 (`rows = 20000`)
1. There are two sample Extended Event session scripts.  One captures logins and the other captures interesting events like errors, slow SQL, blocked processes, mirroring events, and writes to the error log.
1. I suggest capturing the `system_health`, `AlwaysOn_health`, and these two sessions.
1. The `sources` can be broken out into a separate file named `sqlxewriter_sources.toml`.  This makes the editing easier or allows the file to be generated from Puppet or some other tool. 


<a name="whats-new"></a>What's New
------------------------------------------

### Release 1.8
* When running in a container, GOMAXPROCS is set to the CPU quota
* Truncating strings better handles Unicode characters
* Handling `errorlog_written` messages is significantly improved
  * Save the raw message in `errorlog_raw`
  * Don't capture `errorlog_date` and `errorlog_time`.  The date was being forced to UTC which is often wrong.
  * Parse error strings for any error event, not just login failures.  These are stored in `error_number`, `severity`, and `state`.
  * Trim most strings to 8000 characters.  A few such as `errorlog_process` are trimmed to a shorter length.
* For `errorlog_written` and `error_reported` events, if it finds text like `[CLIENT: 10.10.1.1]`, it will save the address in `xe_client_address`
* Update new builds to GO 1.22.5

### Tag 1.7.9
This is a tag only.
* Update to GO 1.19
* Update all dependencies
* The username and password fields for Defaults, Sources, and the Elastic Sink can be set to an environment variable. That looks like this: `password="$(env:VARIABLE_NAME)"`.

### Release 1.7.8
* Improve `errorlog_written` event parsing to populate any `error_number`, `severity`, or `state` that it can parse out
* Add a `sampler` sink (see below).  This writes sample events.  It is primarily used during development or pre-deployment to work on JSON format.
* Improve handling bad offsets in event files.  Try to read past to the next good event in more conditions.
* File Sink: Clean up old event files at the start
* Additional testing against SQL Server on Linux, SQL Server 2022 CTP2, and SQL Server running in Docker
* Set `beta_features` to `true` in the `app` settings section to add the following calculated fields: `cpu_time_sec`, `logical_reads_mb`, `physical_reads_mb`, `writes_mb`, and `duration_sec`.  See the [Beta Features](#beta) section for more details.

### Release 1.7.6

* Capture Availability Groups in `mssql_ag` and Listeners in `mssql_ag_listener`.  These are both multi-value fields.
* Fields can be upper or lower case. Use the `lowercase=["fld1", "fld2"]` to list the fields that should be lower case.  Use `uppercase` for uppercase.
* `hadr_trace_message` events populate the `xe_description` field
* Update license to Internal Use Only license
* Log the time stamp of the configuration files

### Release 1.7.4

* Fix bug that didn't handle adds, moves, etc. in `sqlxewriter_sources.toml` files
* Add support for `attention` events
* Update dependencies
* WSL2 doesn't allow connections to local SQL Server instances (easily).  Since the Linux version can't be fully tested, this doesn't include the binaries.  If you are running on Linux, you will have to [build your own executable](#building).

### Release 1.7.1 

* Add `xe_category` field.  This defaults to the event name but groups similar events together. For example, all SQL events are in `tsql`, all HADR events are in `hadr`, `deadlock`, etc.  This should make it easier to filter events.
* Better handle errors when the state file is all NULLs
* Upgrade GO to 1.16.5
* Improve logging messages
* Big rewrite of logstash sink to improve reliability in the face of disappearing or non-responsive logstash servers
* Retire TLS 1.1 if connecting to Elastic over HTTPS
* Add support for `server_name_override` and `domain_name_override` in sources.  See below for details.

### Release 1.5.1

* Minor memory cleanup

### Release 1.5

* Add limited macOS (Darwin) support. You'll have to compile it yourself.
* Add a `.vscode` directory in the samples with extensions and file mapping support
* Fix the git commit information in the Linux build
* Fix the subdirectory in the Linux build
* Update the README for first-timers
* Add the server name to the polling message (in addition to being in the field)

### Release 1.4.3

* Improves support for trusted connections in Linux.  See [Linux Support](#linux) for more details.
* Added the `login_failed` column.  This is populated for `errorlog_written` (logon) events and error_reported events whose error number indicates a login failed event.  Filter Kibana for the existence of this field.  See below for more details.

### Release 1.4

* We now read past invalid offsets.  The most likely cause of invalid offsets is that this service stopped or the files rolled over so fast that the file and offset are no longer valid.  In both cases, the extended event files rolled over and the events no longer exist. Previously this application would log the error and wait so we could investigate and fix.  Meanwhile, more events were being missed.  Now it just logs the error and tries to find the next valid event for the extended event session.

### Release 1.3.2

* Clean up leaking connection pool for invalid servers
* Update what gets written into the variable substitutions for GIT based on using GORELEASER.  See "Replacements" further down.

### Release 1.3.1

* Add [GOReleaser](https://goreleaser.com/) support for Linux builds

### Release 1.3.0 (Pre-Release)

* Experimental support for Linux and macOS
* The driver library to connect to SQL Server has changed.  Please report any issues.

### Release 1.2.X (Pre-Release)

* Supports adding filters (see Filters in the [Application Settings](#app-settings))
* Parse Error Log Written events into individual fields

### Release 1.0

* Configuration files are split into two.  Sources can now be split out into a separate file named `sqlxewriter_sources.toml`.  **NOTE: This requires that configuration files are named `sqlxewriter.toml` and `sqlxewriter_sources.toml`.**

### Release 0.92
 
 * Coalesce multiple file change events into a single event.  This behaves much better if you're editing the file while it's running.  I still don't suggest this though.
 * Better handle errors where logstash stops responding

### Release 0.91
* I hated the `-once` flag.  If you run it interactively, it now runs once by default.  If you want it to continue running and polling, you'll need to use the `-loop` flag.  Running as a service isn't affected.
* (BETA) If you enable `watch_config` in the TOML file, it will try to reload the configuration in the event of a file change.  It seems that any save is really two writes: one for the file and one set attributes.  So you'll see two reloads.
* I'm not testing `xelogstash.exe` at all.  Please use a previous version if that's what you want.
* The settings to print a summary and a sample of each event have been removed.  Instead, you can write a small number of events to a file sink and review those.

### Release 0.9
* `sqlxewriter.exe` can run as a Windows service
* Added metrics page to see a count of events read and written
* Log memory use every 24 hours
* Added retry logic for sinks
* Logging to a JSON file suitable for Filebeat



<a name="sources"></a>Sources and Defaults
------------------------------------------

Each SQL Server you want to extract events from is called a "source".  You can specify each source using the `[[source]]` header in the TOML file.  (This is an array of sources.)  The only required field for a source is the `fqdn` which is how you connect to the server.

The `[defaults]` section applies to all sources (but can be overridden by the source).  I find it's easiest to set most of your values in this section and then override them for each source.  The "default" section is just a source itself.  __Note: The  _source_ values override the _default_ values except for adds, copies and moves which are discussed below. Those are merged together.__

You can set the following Source fields (or Default fields)

* `fqdn` is the name to connect to for the server.  It can be a host name, a hostname,port, a host\instance name, a static DNS, or an IP.  This value is just dropped into the connection string.
* `user` and `password`.  If you leave this blank or don't specify them it will connect using a trusted connection.  You can set these to an environment variable by `password="$(env:VARIABLE_NAME)"`.  
* `driver` and `odbc_driver` allow different database drivers.  Normally these values are not needed.
  * `driver` is can be either "mssql" (default) or "odbc".  "mssql" is a native GO driver that handles everything except trusted connections on Linux.
  * `odbc_driver` sets the ODBC driver if using ODBC.  The default is the highest installed ODBC driver.  This is likely "ODBC Driver 18 for SQL Server".  This value must match exactly (case-sensitive) to a driver on the system if using ODBC.
* `sessions` is a list of sessions to process.
* `ignore_sessions` says to not process any sessions for this source.  This is mainly useful if you have a list of default sessions but some old SQL Server 2008 boxes that you want to ignore the sessions completely so you can just get the failed agent jobs.
* `rows` is how many events to try and process per session.  It will read this many events and then continue reading until the offset changes.  Omitting this value or setting it to zero will process all rows since it last ran.
* `agentjobs` can be "all", "failed" or "none".  It tries to map the field names to the extended event field names.
* `excludedEvents` is a list of events to ignore.  Both sample configuration files exclude some of the system health events like ring buffer recorded and diagnostic component results. 
* `adds`, `moves`, and `copies` are described in their own section below.
* `strip_crlf` (boolean) will replace common newline patterns with a space. Some logstash configurations don't handle newlines in their JSON.  The downside is that it de-formats SQL and deadlock fields.
* `start_at` and `stop_at` are used to limit the date range of returned events.  __Please be aware this will almost certainly lead to dropped or duplicated events.  It should only be used for testing.__  The date must be in "2018-01-01T13:14:15Z" or "2018-06-01T12:00:00-05:00" and must be enclosed in quotes in the TOML file.
* `look_back` (duration string) will determine how far back to get events.  It's like a relative `start_at`.  `look_back` is a duration string of a decimal number with a unit suffix, such as "24h", "168h" (1 week), or "60m".  Valid time units are "h", "m", or "s".  If both `start_at` and `look_back` are set, it will use the most recent calculated date between the two.
* `exclude_17830` is a boolean that will exclude 17830 errors.  I typically see these from packaged software and can't do much about them.
* `log_bad_xml` is boolean.  This will write the last bad XML parse to a file. 
* `include_dbghelpdll_msg` is a boolean.  Some versions of SQL Server emit a message like `Using 'dbghelp.dll' version '4.0.5'`.  These are now excluded by default.  This setting adds those back in.

* `server_name_override` allows you to override `@@SERVERNAME` and `SERVERPROPERTY('MachineName')` returned by a source.  This is useful for Linux servers inside containers that set long machine names and SQL Server only returns the first 15 characters.  This supports names longer than 15 characters.
* `domain_name_override` allows you to override `DEFAULT_DOMAIN()` returned by a source.  This is useful for Linux servers that don't technically belong to a domain so they can appear to be in one.

## <a name="json"></a>Controlling the JSON
The two fields `timestamp_field_name` and `payload_field_name` are available in the Source and Default sections.  The following examples best illustrate how they work.

### Example 1
All the event fields are at the root level.

```json 
- - - sqlxewriter.toml - - - 

timestamp_field_name = "@timestamp"
payload_field_name = ""

 - - - generates - - - - - 

{
  "@timestamp": "2018-05-08T01:23:45.691Z",
  "client_app_name": "sqlxewriter.exe",
  "client_hostname": "D30",
  ". . . lots of json fields...": "go here",
  "xe_severity_value": 6
}
```

### Example 2
All the event fields are nested inside an "mssql" field.  This is the most common way to run the application.

```json
- - - sqlxewriter.toml - - - 

timestamp_field_name = "event_time"
payload_field_name = "mssql"

- - - generates - - - 

{
  "event_time": "2018-05-08T01:23:49.928Z",
  "mssql": {
    "client_app_name": "sqlxewriter.exe",
    "client_hostname": "D30",
    ". . . lots of json fields...": "go here",
    "xe_severity_value": 6
  }
}

```

## <a name="adds"></a>Add, Copies, and Moves
Adds, moves, and copies give you the opportunity to modify the generated JSON.  All three are arrays with a format of "string1:string2".  

> Note: For these values, any Source overwrites the Default at _the individual key level_.  If both the default and source try to add a key for "environment", it will use the value from the Source.  But if the Default adds a key for "datacenter" and the Source adds a key for "environment", it will add both keys.  Copies and Moves are handled the same way.

* `adds` are "key:value" that will add a key with the specified value
* `copies` are "src:dest" that will copy the value at _src_ to _dest_
* `moves` are "src:dest" that will move the value from _src_ to _dest_

Further, the keys can be nested using a dotted notation.  For example, setting a key at `global.host.name` will nest the value three levels deep.

Consider the following settings:

```toml
timestamp_field_name = "@timestamp"
payload_field_name = "event"

adds =  [   "global.log.vendor:Microsoft",
            "global.log.type:Application",
            "global.log.collector.application:sqlxewriter.exe",
            "global.log.collector.version:'$(VERSION)'",
        ] 

copies = [  "event.mssql_computer:global.host.name",
            "event.mssql_domain:global.host.domain",
            "event.mssql_version:global.log.version"
        ]
```
That results in this event:
```json
{
  "global": {
    "host": {
      "domain": "WORKGROUP",
      "name": "D30"
    },
    "log": {
      "version": "SQL Server 2016 RTM",
      "collector": {
        "version": "0.12",
        "application": "sqlxewriter.exe"
      },
      "type": "Application",
      "vendor": "Microsoft"
    }
  },
  "@timestamp": "2018-05-08T01:24:47.368Z",
  "event": {
    "client_app_name": "xecap.exe",
    "client_hostname": "D30",
    
    "mssql_computer": "D30",
    "mssql_domain": "WORKGROUP",
    "mssql_server_name": "D30\\SQL2016",
    "mssql_version": "SQL Server 2016 RTM",
    "name": "login",
    
    "server_instance_name": "D30\\SQL2016",
    "server_principal_name": "graz",
    "timestamp": "2018-05-08T01:24:47.368Z",
    "xe_acct_app": "graz - xecap.exe",
    
    "xe_session_name": "Demo-Logins2",
    "xe_severity_keyword": "info",
    "xe_severity_value": 6
  }
}

```

Note how the copies are able to "lift" values out of the event and into other parts of the JSON document.  This helps conform to whatever standards are in your ELK environment.

The values that are added can be strings, integers, floats, booleans, or dates.  Putting a value in single ticks forces it to be a string.

### Replacements
The adds, moves, and copies also support a few "replacement" values.  

* `$(VERSION)` is the version of sqlxewriter.exe.  Note that $(VERSION) should be forced to a string by enclosing it in single quotes in the TOML file.  Otherwise the parser will try to make it a number.
* (DEPRECATED) `$(GITDESCRIBE)` is the same as `$(VERSION)`.  It used to hold the results of `git describe` but GORELEASER does it differently so these are now the same. _This is deprecated and will be removed at some point.  Use VERSION and GITHASH._
* `$(GITHASH)` is the seven character GIT hash that was used for the build
* `$(EXENAMEPATH)` is the full path and name of the executable
* `$(EXENAME)` is the name of the executable
* `$(PID)` is the Process ID of sqlxewriter.exe
* `$(HOST)` is the computer where sqlxewriter.exe is running
* `$(NOW)` is the time that sqlxewriter.exe wrote this value to a sink

See the section below on derived fields for a description of the "mssql_" and "xe_" fields

### Upper and Lower Case Fields
SQL Server generally returns fields in a consistent case.  However I've started to see `@@SERVERNAME` returning lower case on some servers.  Since Elastic Search is case-sensitive this can be challenging. Additionally certain fields my better in upper or lower case.  Field case can be controlled using these fields:

```toml
uppercase = ["mssql.server_name", "mssql.server_princpal_name"]
lowercase = ["global.host.name"]
```

These can be set in the `defaults` section of the TOML file and in each individual source.  The source specific entries are processed after the defaults.  

These are processed after adds, moves, and copies.

## <a name="prefixes"></a>Prefixes and keeping your place

The application keeps track how far it has read into the extended event file target using a state file.  This file holds the file name and offset of each read for that session.  The file is named `Domain_ServerName_Session.state`.  There is also a ".0" file that is used while the application is running.  You can tell the application to start all over by deleting the state file.  The "ServerName" above is populated by `@@SERVERNAME` from the instance.

## <a name="app-settings"></a>Application Settings
These are the fields you can set in the `[app]` section of the configuration file.

### `[app]` section
This controls the overall application.  All these fields are optional.
* `http_metrics` enables a local web server that can provide diagnostic information.  This defaults to false.  It exposes the following URLs:
  * [http://localhost:8080/debug/metrics](http://localhost:8080/debug/metrics) displays counts of events read and written as well as memory usage.
  * [http://localhost:8080/debug/vars](http://localhost:8080/debug/vars) provides some basic metrics in JSON format including the total number of events processed. This information is real-time.
  * [http://localhost:8080/debug/pprof/](http://localhost:8080/debug/pprof/) exposes the [GO PPROF](https://golang.org/pkg/net/http/pprof/) web page for diagnostic information on the executable including memory usage, blocking, and running GO routines.  
* `http_metrics_port` is the port the metrics URLs are exposed on.  It defaults to 8080.  
* `watch_config` (BETA) attempts to stop and restart if the TOML configuration file changes.  This defaults to false.
> Internet Explorer pre-Chromium is horrible for viewing `vars` and `pprof`.  I suggest a newer browser.

### Filters

A series of filters can be added the the TOML configuration file.  That looks like this:

```toml 
[[filter]]
filter_action = "exclude"
error_number = 15151

[[filter]]
filter_action = "include"
error_number = 15151
server_instance_name = "server01"
``` 
The `filter_action` must be `include` or `exclude`.  Each filter that matches ALL fields sets the `action` based on the that value. After all filters have processed, the event is either included or excluded.

For example, this means that a broad filter could exclude an event but a later more specific filter could change the action to include it.

In the example above, all 15151 errors are excluded except for "server01".

## <a name="derived-fields"></a>Derived Fields
Based on a particular event, the application computes a number of calculated fields and adds those to the event.  Most of them have an "xe_" prefix to separate them.  It also returns a few SQL Server level settings with an "mssql_" prefix.

* `mssql_domain`: This is the result of DEFAULT_DOMAIN() run on the source server
* `mssql_computer`: This is the result of SERVERPROPERTY('MachineName') on the source computer.
* `mssql_server_name`: This is the result of @@SERVERNAME on the source computer
* `mssql_version`: Is something like "SQL Server 2016 SP1" composed of various server property attributes.  
* `mssql_product_version`: is SERVERPROPERTY('ProductVersion') which is *major.minor.build.revision* (example: 13.0.5366.1)
* `xe_severity_value`: 3, 5, or 6 for ERR, WARN, INFO based on the syslog values
* `xe_severity_keyword`: "err", "warning", "info" based on the syslog values
* `xe_description`: a text description of the event.  The format
depends on the type of event and what fields are available.  My goal is that seeing the name of the server, the event type (`name` field), and this field are enough to know what happened
* `xe_acct_app`: a combination of the server_principal_name and client_app_name in "acct - app" format.
* `xe_acct_app_client`: a combination of the server_principal_name, client_app_name, and client_hostname in "acct - app (client)" format
* `xe_session_name`: name of the extended event session for this event
* `xe_file_name`: name of the XE file for this event
* `xe_file_offset`: file offset where we found this event
* `xe_category`: defaults to the event name but groups similar events together. For example, all SQL events are in `tsql`, all HADR events are in `hadr`, `deadlock`, etc.
* `server_instance_name`: This is normally provided by the extended event.  However system_health and AlwaysOn_health don't capture this.  If the field isn't provided, I populate it from `@@SERVERNAME` of the source server.
* `login_failed`: This field is populated when a login fails.  The easiest way to monitor failed logins in Kibana is filter for the existence of the `login_failed` field. Login errors are reported two ways and this tries to captures both.  That means you will typically see two errors in Kibana for each failed attempt.
  * For `errorlog_written`, if the errorlog written process is `logon` it populates this field with the error message.  That has the IP address of the client.  It also means that if you're capturing successful logins in the error log, this will be wrong.  Successful logins should be captured by extended events.
  * For `error_reported`, if the error number is one whose text has "login failed", then we populate the field with the error message.


## <a name="sinks"></a>Sinks
XEWriter can write to multiple targets called "sinks".  It can write to files, to logstash, or directly to Elastic Search.  It can write to all three sinks at the same time if they are all specified.  They are written serially so the performance isn't that great.

### File Sink
This is configured with a `filesink` section:

````toml
[filesink]
retain_hours = 24
````
The files are named for the server, instance, session name, date, and hour and are written to an `events` directory with a `.json` extension.  The files are rotated every hour.  Any files older than  `retain_hours` are removed.

These files should be imported into Logstash or Elastic Search using [FileBeat](https://www.elastic.co/products/beats/filebeat).  This is probably the simplest way to import.

### Logstash Sink
This is configured with a `logstash` section:

```toml
[logstash]
host = "localhost:8888"
```

This writes the events directly to the specified logstash server.

### Elastic Sink
This is configured using the `elastic` section. This is the most complicated to configure.

```toml
[elastic]
addresses = ["https://host.domain.com:1234"]
username = "dev-user"
password = "horsebatterysomethingsomething"

auto_create_indexes = true
default_index = "dev-sql"

event_index_map = [
    "login:dev-login"
]
```

* `addressess` specifies one or more addresses for the Elastic servers.
* `username` and `password` provide authentication.  You can set these to an environment variable by `password="$(env:VARIABLE_NAME)"`.
* `auto_create_indexes` controls whether the application tries to create indexes.  
* `default_index` is the index where events will be written unless overridden by the event index map.
* `event_index_map` allows mapping different events to different indexes.  In the example above, all the events except `login` will go to the `dev-sql` index.  The `login` events will go to the `dev-login` index.  I often split login event into their own index.

### Sampler Sink
This is configured using the `sampler` section.  This writes sample events for review. It is primarily used in development.  It writes one file per extended event type.  The files are located in `./sinks/sampler`.  

```toml
[sampler]
duration = "1m"
```

This will write an event type every one minute.  The duration is a sequence of decimal numbers, each with a unit suffix, without spaces.  Valid units are "h", "m", "s".  Examples include "10m", "1h30m", "1m", "10s".

## <a name="beta"></a>Beta Features
Enabling `beta_features` in the `app` section will enable certain beta features:

```toml
[app]
beta_features = true
```
As of 1.7.8, this adds the following calculated fields: `cpu_time_sec`, `logical_reads_mb`, `physical_reads_mb`, `writes_mb`, and `duration_sec`.
These fields are only added if the quantity at or above the threshold.  For example, `cpu_time_sec` is only added if the CPU time is at least 1 second (1 million microseconds).

## <a name="linux"></a>Linux Support

Experimental support is included for Linux.  Please be aware of the following issues:

* This was only tested using WSL2 to connect to a local SQL Server
* Trusted connections only work using the ODBC driver.  Set the following for the "default" section or for each source:
  * `driver="odbc"`
  * `odbc_driver="ODBC Driver 17 for SQL Server"` or the ODBC driver you are using.  This is the only one I've tested with.
* Reloading configuration on file change doesn't seem to work
* macOS doesn't compile properly at this time.


## <a name="notes"></a>Other Notes

1. I find that setting the `rows = 20000` in the `[defaults]` section works well.  It's enough rows that it catches up quickly if I pause the job.  

2. The sources are processed in the order they are listed.  Each server is polled every minute.  It spreads out the servers evenly over the minute.

3. I make some decisions around setting the severity level.  Failed jobs and job steps are errors.  SQL Server errors are errors.  I haven't gone much beyond that yet.

4. I haven't done much in the way of optimizations yet.  It will process between 2,000 and 3,000 events per second on my aging desktop with SQL Server running on the same box.  A properly scaled Logstash doesn't slow it down much.  I have a few servers that keep 1 GB of login events in 50 MB files.  It takes roughly 20 minutes to get through it the first time.

5. If it gets behind and the offset becomes invalid, it will log an error.  It will also set a flag to try and catch up next time.  That flag is a third column in the status file that says "reset".  If it finds that, it will start at the beginning of the extended event file target and read until it gets to an event after the last one it saw.  It will also log an error that events were probably skipped.

## <a name="building"></a>Building the Application

* The application is currently built with GO 1.16.5
* The tests can be run with `go test ./...`
* Security scanning can be done by `gosec -severity medium ./...`

* The builds write to a `deploy` directory
  * Build for Windows by running `PSMake.cmd 1.1.1` (or the target version)
  * Build for Linux by running `./build.sh 1.1.1` from Linux.  I use WSL2 with Ubuntu 20.04.  We use the ODBC driver which uses CGO.  Cross-platform builds don't work well.
* SQLXEWriter can be built directly with `go build ./cmd/sqlxewriter`

