/*
This is a sample extended event session to capture non-pooled logins
*/

SET XACT_ABORT ON;

-- Stop the session
IF EXISTS (select * from sys.dm_xe_sessions WHERE [name] = 'logstash_logins')
  BEGIN
	PRINT 'Stopping session...';
	ALTER EVENT SESSION logstash_logins ON SERVER STATE = STOP;
  END

-- Drop the session 
IF EXISTS (select * from sys.server_event_sessions WHERE [name] = 'logstash_logins')
  BEGIN
	PRINT 'Dropping session...';
	DROP EVENT SESSION logstash_logins ON SERVER;
  END

-- Create the session
PRINT 'Creating the session...';

CREATE EVENT SESSION [logstash_logins] ON SERVER 
ADD EVENT sqlserver.login(
    ACTION(
			package0.event_sequence,
			sqlserver.client_app_name,
			sqlserver.client_hostname,
			sqlserver.client_pid,
			sqlserver.database_name,
			sqlserver.server_instance_name,
			sqlserver.server_principal_name,
			sqlserver.session_id)
    WHERE ([is_cached]=(0)))
ADD TARGET package0.event_file(
	SET filename=N'logstash_logins',max_file_size=(50),max_rollover_files=(10))
WITH (	MAX_MEMORY=4096 KB,
		EVENT_RETENTION_MODE=ALLOW_SINGLE_EVENT_LOSS,
		MAX_DISPATCH_LATENCY=10 SECONDS,
		STARTUP_STATE=ON)
GO


-- Start the session
IF NOT EXISTS (select * from sys.dm_xe_sessions WHERE [name] = 'logstash_logins')
  BEGIN
	PRINT 'Starting session...';
	ALTER EVENT SESSION logstash_logins ON SERVER STATE = START;
  END


