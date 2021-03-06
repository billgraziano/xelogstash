# XELogstash

> **`xelogstash.exe` is deprecated.  Please use the newer `xewriter.exe`.**
> **It is only included for backward compatibility.** 

`xelogstash.exe` is a command-line application to pull SQL Server Extended Events and SQL Server Agent job results and push them to Logstash.   It supports SQL Server 2012 and higher.  It is untested against SQL Server 2008 (R2).

1. [Breaking Changes](#breaking-changes)
1. [Getting started](#getting-started)
2. [Sources and Defaults](#sources)
2. [Controlling JSON](#json)
1. [Adds, Moves, Copies](#adds)
2. [Prefixes and keeping your place](#prefixes)
2. [Application Settings](#app-settings)
3. [Derived Fields](#derived-fields)
3. [Sinks](#sinks)
4. [Other Notes](#notes)

## <a name="getting-started"></a>Breaking Changes
* post-0.44 - The AppLog section has been removed.  Please remove this from your TOML files.  All application log files are now written to JSON.
* post-0.44 - The application log files are written to the `log` subdirectory in JSON format.   These should be processed with FileBeat.
* 0.40 - The TOML configuration to write to logstash has moved.  It was formerly under `app.logstash`.  Now it is configured in the `logstash` section.  See [Sinks](#sinks) below.

## <a name="getting-started"></a>Getting Started
The application uses a [TOML](https://en.wikipedia.org/wiki/TOML) file for configuration.  Two sample files are included. 

1. Extract the ZIP contents to a directory.  We'll be starting with "start.toml".
2. If you have a local SQL Server installed, no changes are necessary.
3. Otherwise, edit the `fqdn` under ``[[source]]`` to point to a SQL Server
4. From a command-prompt, type "`xelogstash.exe`".  (This doesn't send anything to logstash yet)

This should generate a `samples.xe.json` and an `xestate` folder.  The `samples.xe.json` file is one of each type of event that would have been sent to Logstash.  This gives you a chance to see how things will look.  The `xestate` folder is used to keep track of the read position in the XE session.  

> NOTE: The permissions on the `xestate` directory are limited. When switching to a service account, be prepared to reset the permissions or delete the directory and allow the service account to recreate it.

### Writing events to a file
In `xelogstash.toml`, uncomment the two lines of the `filesink` section and rerun the application.  This will write events to a file in the `events` directory in JSON format.  Each source server XE session gets a file and they rotate every hour.  These files can be written to Elastic Search using [FileBeat](https://www.elastic.co/products/beats/filebeat).

### Sending to Logstash
To send events to directly Logstash, specify the `logstash` section with a `host` value.  The `host` should be in `host:port` format.  After that you can run the executable with the same parameters. 

````toml
[logstash]
host = "localhost:8888"
````

### Command Line Options 
There are three command line options.  Running `xelogstash /?` will display the help for the options.

- `/config filename` - Tries to load the TOML file from _filename_. If this isn't specified, `xelogstash.toml` is used.
- `/log` - Captures the application output to a log file INSTEAD of standard out.  The log files are located in the `log` subdirectory and named `xelogstash_YYYYMMDD.log` based on the start time.  Log files are automatically deleted after 7 days and that's not configurable yet. 
- `/debug` - Enables additional debugging output.


## <a name="sources"></a>Sources and Defaults
Each SQL Server you want to extract events from is called a "source".  You can specify each source using the `[[sourece]]` header in the TOML file.  (This is an array of sources.)  The only required field for a source is the `fqdn` which is how you connect to the server.

The `[defaults]` section applies to all sources (but can be overridden by the source).  I find it's easiest to set most of your values in this section and then override them for each source.  The "default" section is just a source itself.  __Note: The  _source_ values override the _default_ values except for adds, copies and moves which are discussed below. Those are merged together.__

You can set the following Source fields (or Default fields)

* `fqdn` is the name to connect to for the server.  It can be a host name, a hostname,port, a host\instance name, a static DNS, or an IP.  This value is just dropped into the connection string.
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


## <a name="json"></a>Controlling the JSON
The two fields `timestamp_field_name` and `payload_field_name` are available in the Source and Default sections.  The following examples best illustrate how they work.

### Example 1
All the event fields are at the root level.

```json 
- - - xelogstash.toml - - - 

timestamp_field_name = "@timestamp"
payload_field_name = ""

 - - - generates - - - - - 

{
  "@timestamp": "2018-05-08T01:23:45.691Z",
  "client_app_name": "xecap.exe",
  "client_hostname": "D30",
  ". . . lots of json fields...": "go here",
  "xe_severity_value": 6
}
```

### Example 2
All the event fields are nested inside an "mssql" field.

```json
timestamp_field_name = "event_time"
payload_field_name = "mssql"
----------------------
{
  "event_time": "2018-05-08T01:23:49.928Z",
  "mssql": {
    "client_app_name": "xecap.exe",
    "client_hostname": "D30",
    ". . . lots of json fields...": "go here",
    "xe_severity_value": 6
  }
}

```

## <a name="adds"></a>Add, Copies, and Moves
Adds, moves, and copies give you the opptunity to modify the generated JSON.  All three are arrays with a format of "string1:string2".  

Note: For these values, any Source overwrites the Default at _the individual key level_.  If both the default and source try to add a key for "environment", it will use the value from the Source.  But if the Default adds a key for "datacenter" and the Source adds a key for "environment", it will add both keys.  Copies and Moves are handled the same way.

* `adds` are "key:value" that will add a key with the specified value
* `copies` are "src:dest" that will copy the value at _src_ to _dest_
* `moves` are "src:dest" that will move the value from _src_ to _dest_

Further, the keys can be nested using a dotted notation.  For example, setting a key at `global.host.name` will nest the value three levels deep.

Consider the following settings:

```json 
timestamp_field_name = "@timestamp"
payload_field_name = "event"

adds =  [   "global.log.vendor:Microsoft",
            "global.log.type:Application",
            "global.log.collector.application:xelogstash.exe",
            "global.log.collector.version:'$(VERSION)'",
        ] 

copies = [  "event.mssql_computer:global.host.name",
            "event.mssql_domain:global.host.domain",
            "event.mssql_version:global.log.version"
        ]
----------------------------
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
        "application": "xelogstash.exe"
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

* `$(VERSION)` is the version of xelogstash.exe.  Note that $(VERSION) is forced to a string by enclosing it in single ticks.
* `$(GITDESCRIBE)` is Git Describe from the build.
* `$(EXENAMEPATH)` is the full path and name of the executable
* `$(EXENAME)` is the name of the executable
* `$(PID)` is the Process ID of xelogstash.exe
* `$(HOST)` is the computer where xelogstash.exe is running
* `$(NOW)` is the time that xelogstash.exe wrote this value to a sink

See the section below on derived fields for a description of the "mssql_" and "xe_" fields

## <a name="prefixes"></a>Prefixes and keeping your place
**NOTE**: Starting in v0.20, the /status directory is being moved to the /xestate field.  This should be done for you by the application.  The new file name format is `domain_instance_class_identifier.state`.  Please leave the Prefix in the TOML file for now.  It will be removed in a future release.

The application keeps track how far it has read into the extended event file target using a state file.  This file holds the file name and offset of each read for that session.  The file is named `Domain_ServerName_Session.state`.  There is also a ".0" file that is used while the application is running.  You can tell the application to start all over by deleting the state file.  The "ServerName" above is populated by `@@SERVERNAME` from the instance.

## <a name="app-settings"></a>Application Settings
These are the fields you can set in the `[app]` section of the configuration file.

### `[app]` section
This controls the overall application.  All these fields are optional.
* `logstash` is the address of the Logstash server is _host:port_ format.  If empty, it will not send events to logstash.
* `samples` set to true will save a JSON file with one of each event type that was processed.  This is very helpful for testing your JSON format.
* `summary` set to true will print a nice summary of the output including how many of each type of event were processed.
* `workers` controls how many concurrent workers will process the sources.  It defaults to the number of cores in the computer.  A given worker will process all the sessions for a source before moving on to the next source.  The application doesn't use much CPU.  It spends lots of time waiting on data to return from sources.
* `http_metrics` enables a local web server that can provide diagnostic information.  This defaults to false.  It exposes the following two URLs:
  * [http://localhost:8080/debug/vars](http://localhost:8080/debug/vars) provides some basic metrics in JSON format including the total number of events processed. This information is real-time.
  * [http://localhost:8080/debug/pprof/](http://localhost:8080/debug/pprof/) exposes the [GO PPROF](https://golang.org/pkg/net/http/pprof/) web page for diagnostic information on the executable including memory usage, blocking, and running GO routines.  
  * IE is horrible for viewing these.  I've found that using PowerShell and running `Invoke-RestMethod "http://localhost:8080/debug/vars"` works well to view that URL



### `[applog]` section
Note: This section was removed post 0.44.



## <a name="derived-fields"></a>Derived Fields
Based on a particular event, the application computes a number of calculated fields and adds those to the event.  Most of them have an "xe_" prefix to separate them.  It also returns a few SQL Server level settings with an "mssql_" prfix.

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
* `server_instance_name`: This is normally provided by the extended event.  However system_health and AlwaysOn_health don't capture this.  If the field isn't provided, I populate it from @@SERVERNAME of the source server.

## <a name="sinks"></a>Sinks
xelogstash can write to multiple targets called "sinks".  It can write to files, to logstash, or directly to Elastic Search.  It can write to all three sinks at the same time if they are all specified.  They are written serially so the performanc isn't that great.

### File Sink
This is configured with a `filesink` section:

````toml
[filesink]
retain_hours = 1 
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
````

* `addressess` specifies one or more addresses for the Elastic servers.
* `username` and `password` provide authentication.
* `auto_create_indexes` controls whether the application tries to create indexes.  It creates default indexes.
* `default_index` is the index where events will be written unless overridden by the event index map.
* `event_index_map` allows mapping different events to different indexs.  In the example above, all the events except `login` will go to the `dev-sql` index.  The `login` events will go to the `dev-login` index.  I often split login event into their own index.

## <a name="notes"></a>Other Notes
1. I've had issues with the SQL Server Agent job ending but not stopping the executable when I manually stop the job.  The application now sets a lock file so that a second instance will exit with an error.   The lock file name is based on the TOML file name (`TOML_file_name.lock`).  Find the first instance in Task Manager and kill it.  I usually only see this if I stop it in the middle of a run or Logstash is behaving oddly.

1. I find that setting the `rows = 20000` in the `[defaults]` section works well.  It's enough rows that it catches up quickly if I pause the job.  It's not so many that adding a new server runs for 20 minutes and everything else pauses.

2. The sources are processd in the order they are listed.

3. I make some decisions around setting the severity level.  Failed jobs and job steps are errors.  SQL Server errors are errors.  I haven't gone much beyond that yet.

4. I haven't done much in the way of optimizations yet.  It will process between 2,000 and 3,000 events per second on my aging desktop with SQL Server running on the same box.  A properly scaled Logstash doesn't slow it down much.  I have a few servers that keep 1 GB of login events in 50 MB files.  It takes roughly 20 minutes to get through it the first time.

5. If it gets behind and the offset becomes invalid, it will log an error.  It will also set a flag to try and catch up next time.  That flag is a third column in the status file that says "reset".  If it finds that, it will start at the beginning of the extended event file target and read until it gets to an event after the last one it saw.  It will also log an error that events were probably skipped.
