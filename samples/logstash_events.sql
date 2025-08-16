/*
This is a sample extended event session 
to capture interesting events for ELK
*/

SET XACT_ABORT ON;

-- Stop the session
IF EXISTS (select * from sys.dm_xe_sessions WHERE [name] = 'logstash_events')
  BEGIN
	PRINT 'Stopping logstash_events...';
	ALTER EVENT SESSION [logstash_events] ON SERVER STATE = STOP;
  END

-- Drop the session 
IF EXISTS (select * from sys.server_event_sessions WHERE [name] = 'logstash_events')
  BEGIN
	PRINT 'Dropping logstash_events...';
	DROP EVENT SESSION [logstash_events] ON SERVER;
  END

-- Create the session
PRINT 'Creating logstash_events...';

CREATE EVENT SESSION [logstash_events] ON SERVER 

ADD EVENT sqlserver.blocked_process_report(
    ACTION(sqlserver.client_app_name,sqlserver.client_hostname,sqlserver.client_pid,sqlserver.database_name,sqlserver.query_hash,sqlserver.server_instance_name,sqlserver.server_principal_name,sqlserver.session_id,sqlserver.sql_text)),

ADD EVENT sqlserver.error_reported(
    ACTION(sqlserver.client_app_name,sqlserver.client_hostname,sqlserver.client_pid,sqlserver.database_name,sqlserver.query_hash,sqlserver.server_instance_name,sqlserver.server_principal_name,sqlserver.session_id,sqlserver.sql_text)
    WHERE (
		-- Severity >= 11 
		( [package0].[greater_than_equal_int64]( [severity],(11))) 
		
		-- but not these errors
		AND [package0].[not_equal_int64]([error_number],(2557))  -- user can't run DBCC (often when using linked servers)
		AND [package0].[not_equal_int64]([error_number],(17830)) -- network error connecting, usually transient
		AND [package0].[not_equal_int64]([error_number],(9104))) -- auto statistics internal 
		),

ADD EVENT sqlserver.lock_deadlock_chain(SET collect_database_name=(1),collect_resource_description=(1)
    ACTION(sqlserver.client_app_name,sqlserver.client_hostname,sqlserver.client_pid,sqlserver.database_name,sqlserver.plan_handle,sqlserver.query_hash,sqlserver.query_plan_hash,sqlserver.server_instance_name,sqlserver.server_principal_name,sqlserver.session_id,sqlserver.sql_text)),

	-- All the SQL events capture for cpu_time > 1 second OR logical reads >- 500,000 pages

ADD EVENT sqlserver.rpc_completed(SET collect_statement=(1)
    ACTION(sqlserver.client_app_name,sqlserver.client_hostname,sqlserver.client_pid,sqlserver.database_name,sqlserver.query_hash,sqlserver.server_instance_name,sqlserver.server_principal_name,sqlserver.session_id,sqlserver.sql_text)
    WHERE ([cpu_time] >= 1000000 OR [logical_reads]>=500000)),

ADD EVENT sqlserver.sp_statement_completed(SET collect_object_name=(1)
    ACTION(sqlserver.client_app_name,sqlserver.client_hostname,sqlserver.client_pid,sqlserver.database_name,sqlserver.query_hash,sqlserver.server_instance_name,sqlserver.server_principal_name,sqlserver.session_id,sqlserver.sql_text)
    WHERE ([cpu_time] >= 1000000 OR [logical_reads]>=500000)),

ADD EVENT sqlserver.sql_batch_completed(SET collect_batch_text=(1)
    ACTION(sqlserver.client_app_name,sqlserver.client_hostname,sqlserver.client_pid,sqlserver.database_name,sqlserver.query_hash,sqlserver.server_instance_name,sqlserver.server_principal_name,sqlserver.session_id,sqlserver.sql_text)
    WHERE ([cpu_time]>=(1000000) OR [logical_reads]>=500000)),

ADD EVENT sqlserver.sql_statement_completed(SET collect_parameterized_plan_handle=(1)
    ACTION(sqlserver.client_app_name,sqlserver.client_hostname,sqlserver.client_pid,sqlserver.database_name,sqlserver.query_hash,sqlserver.server_instance_name,sqlserver.server_principal_name,sqlserver.session_id,sqlserver.sql_text)
    WHERE ([cpu_time] >= 1000000 OR [logical_reads]>=500000)),

ADD EVENT sqlserver.xml_deadlock_report(
    ACTION(sqlserver.client_app_name,sqlserver.client_hostname,sqlserver.client_pid,sqlserver.database_name,sqlserver.query_hash,sqlserver.server_instance_name,sqlserver.server_principal_name,sqlserver.session_id,sqlserver.sql_text)),

ADD EVENT sqlserver.database_mirroring_state_change(
    ACTION(package0.event_sequence,sqlserver.client_app_name,sqlserver.client_hostname,sqlserver.client_pid,sqlserver.database_name,sqlserver.is_system,sqlserver.server_instance_name,sqlserver.server_principal_name,sqlserver.sql_text)),

ADD EVENT sqlserver.errorlog_written(
    ACTION(package0.event_sequence,sqlserver.client_app_name,sqlserver.client_hostname,sqlserver.client_pid,sqlserver.database_name,sqlserver.is_system,sqlserver.server_instance_name,sqlserver.server_principal_name,sqlserver.sql_text)),

ADD EVENT sqlserver.database_file_size_change(SET collect_database_name=(1)
    ACTION(sqlserver.client_app_name,sqlserver.client_hostname,sqlserver.database_name,sqlserver.server_instance_name,sqlserver.server_principal_name,sqlserver.sql_text))

ADD TARGET package0.event_file(
	SET filename=N'logstash_events',max_file_size=(10), max_rollover_files=(20)  )

--ADD TARGET package0.ring_buffer(
--	SET max_memory=(1024))

WITH (
	MAX_MEMORY=4096 KB,
	EVENT_RETENTION_MODE=ALLOW_SINGLE_EVENT_LOSS,
	MAX_DISPATCH_LATENCY=10 SECONDS,
	STARTUP_STATE=ON)
GO

-- Start the session
IF NOT EXISTS (select * from sys.dm_xe_sessions WHERE [name] = 'logstash_events')
  BEGIN
	PRINT 'Starting logstash_events...';
	ALTER EVENT SESSION [logstash_events] ON SERVER STATE = START;
  END

