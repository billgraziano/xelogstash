# xelogstash
Pull SQL Server Extended Events and push them to Logstash

This supports SQL Server 2012 and higher.  It is untested against SQL Server 2008 (R2).

1. Getting started
2. Sources and Defaults
1. SQL Server Agent Jobs
2. Prefixes and keeping your place 
2. Application Settings
2. Controlling JSON
3. [Derived Fields](#derived-fields)
4. Substitutions

## <a name="derived-fields"></a>Derived Fields
Based on a particular event, it computes a number of calculated fields and adds those to the event.  Most of them
have an "xe_" prefix to separate them.

* `mssql_domain`: This is the result of DEFAULT_DOMAIN() run on the source server
* `mssql_computer`: This is the result of SERVERPROPERTY('MachineName') on the source computer.
* `mssql_server_name`: This is the result of @@SERVERNAME on the source computer
* `mssql_version`: Is something like "SQL Server 2016 SP1" composed of various server property attributes.  
* `xe_severity_value`: 3, 5, or 6 for ERR, WARN, INFO based on the syslog values
* `xe_severity_keyword`: err, warn, info based on the syslog values
* `xe_description`: a text description of the event.  The format
depends on the type of event and what fields are available.
* `xe_acct_app`: a combination of the server_principal_name and client_app_name in "acct - app" format.
* `xe_acct_app_client`: a combination of the server_principal_name, client_app_name, and client_hostname in "acct - app (client)" format
* `xe_session_name`: name of the extended event session for this event
* `xe_file_name`: name of the XE file for this event
* `xe_file_offset`: file offset where we found this event
* `server_instance_name`: This is normally provided by the extended event.  However system_health and AlwaysOn_health don't capture this.  If the field isn't provided, I populate it from @@SERVERNAME of the source server.



(Document the other fields I calc here)