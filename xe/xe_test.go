package xe

import (
	"encoding/json"
	"encoding/xml"
	"testing"
	"time"
)

var i SQLInfo

func init() {

	i = SQLInfo{
		Server:         "D30",
		Domain:         "WORKGROUP",
		Computer:       "D30",
		ProductLevel:   "Test",
		ProductRelease: "Test",
		Version:        "13.0",
	}

}

var loginEventData = `
<event name="login" package="sqlserver" timestamp="2018-04-08T16:00:53.427Z">
	<data name="is_cached"><value>false</value></data>
	<data name="is_dac"><value>false</value></data>
	<data name="database_id"><value>1</value></data>
	<data name="packet_size"><value>4096</value></data>
	<data name="options"><value>2000002838f4010000000000</value></data>
	<data name="options_text"><value><![CDATA[]]></value></data>
	<data name="database_name"><value><![CDATA[]]></value></data>
	
	<action name="client_app_name" package="sqlserver"><value><![CDATA[IsItSQL]]></value></action>
	<action name="client_hostname" package="sqlserver"><value><![CDATA[D30]]></value></action>
	<action name="client_pid" package="sqlserver"><value>12036</value></action>
	<action name="database_name" package="sqlserver"><value><![CDATA[master]]></value></action>
	<action name="server_instance_name" package="sqlserver"><value><![CDATA[D30\SQL2012]]></value></action>
	<action name="server_principal_name" package="sqlserver"><value><![CDATA[D30\Bill]]></value></action>
</event>
`

func TestXMLParsing(t *testing.T) {
	var ed XMLEventData
	err := xml.Unmarshal([]byte(loginEventData), &ed)
	if err != nil {
		t.Error(err)
	}
	if ed.Name != "login" {
		t.Error("Invalid Name", ed.Name)
	}
	if ed.TimeStamp.Day() != 8 {
		t.Error("Bad Date Parse", ed.TimeStamp)
	}
}

func TestBasicParse(t *testing.T) {
	event, err := Parse(&i, loginEventData)
	if err != nil {
		t.Error(err)
	}

	v, ok := event["name"]
	if !ok {
		t.Error("name not found")
	}
	name, ok := v.(string)
	if !ok {
		t.Error("name isn't a string??")
	}
	if name != "login" {
		t.Error("name not login")
	}

	iface, ok := event["timestamp"]
	if !ok {
		t.Error("timestamp not found")
	}

	ts, ok := iface.(time.Time)
	if !ok {
		t.Error("timestamp isn't a time")
	}
	if ts.Day() != 8 {
		t.Error("Wrong timestamp")
	}

	//t.Log(event.print())
}

func TestJson(t *testing.T) {
	event, err := Parse(&i, loginEventData)
	if err != nil {
		t.Error(err)
	}
	//t.Log("Raw Event: ", event)
	_, err = json.Marshal(event)
	if err != nil {
		t.Error(err)
	}
	// t.Log(jsonBytes)
	//jsonString := string(jsonBytes)
	//t.Log("JSON String: ", jsonString)
}

func TestErrorLogEvent(t *testing.T) {

	rawXML := `
	<event 
		name="errorlog_written" package="sqlserver" 
		timestamp="2018-02-04T00:42:25.276Z">
		
		<data name="message"><value><![CDATA[2018-02-03 18:42:25.28 spid4s      SQL Trace ID 1 was started by login "sa".  ]]></value></data>
		
		<action name="session_id" package="sqlserver"><value>4</value></action>
		<action name="server_principal_name" package="sqlserver"><value><![CDATA[sa]]></value></action>
		<action name="server_instance_name" package="sqlserver"><value><![CDATA[D30\SQL2016]]></value></action>
		<action name="is_system" package="sqlserver"><value>true</value></action>
		<action name="database_name" package="sqlserver"><value><![CDATA[master]]></value></action>
		<action name="client_pid" package="sqlserver"><value>0</value></action>
		<action name="client_hostname" package="sqlserver"><value><![CDATA[]]></value></action>
		<action name="client_app_name" package="sqlserver"><value><![CDATA[]]></value></action>
		<action name="event_sequence" package="package0"><value>1</value></action>
		</event>
	`

	event, err := Parse(&i, rawXML)
	if err != nil {
		t.Error(err)
	}
	t.Log("Raw Event: ", event)
	jsonBytes, err := json.Marshal(event)
	if err != nil {
		t.Error(err)
	}
	// t.Log(jsonBytes)
	jsonString := string(jsonBytes)
	t.Log("JSON String: ", jsonString)
}

func TestErrorReportedEvent(t *testing.T) {

	rawXML := `<event name="error_reported" package="sqlserver" timestamp="2018-04-19T16:21:25.541Z"><data name="error_number"><value>208</value></data><data name="severity"><value>16</value></data><data name="state"><value>1</value></data><data name="user_defined"><value>false</value></data><data name="category"><value>2</value><text><![CDATA[SERVER]]></text></data><data name="destination"><value>0x00000002</value><text><![CDATA[USER]]></text></data><data name="is_intercepted"><value>false</value></data><data name="message"><value><![CDATA[Invalid object name 'sys.xe_object_columns'.]]></value></data><action name="sql_text" package="sqlserver"><value><![CDATA[select * from sys.xe_object_columns ]]></value></action><action name="server_principal_name" package="sqlserver"><value><![CDATA[MicrosoftAccount\graz]]></value></action><action name="server_instance_name" package="sqlserver"><value><![CDATA[D30\SQL2016]]></value></action><action name="is_system" package="sqlserver"><value>false</value></action><action name="database_name" package="sqlserver"><value><![CDATA[master]]></value></action><action name="client_hostname" package="sqlserver"><value><![CDATA[D30]]></value></action><action name="client_app_name" package="sqlserver"><value><![CDATA[Microsoft SQL Server Management Studio - Query]]></value></action><action name="collect_system_time" package="package0"><value>2018-04-19T16:21:25.540Z</value></action></event>`

	event, err := Parse(&i, rawXML)
	if err != nil {
		t.Error(err)
	}
	t.Log("Raw Event: ", event)
	jsonBytes, err := json.Marshal(event)
	if err != nil {
		t.Error(err)
	}
	// t.Log(jsonBytes)
	jsonString := string(jsonBytes)
	t.Log("JSON String: ", jsonString)
}

func TestProcStatemt(t *testing.T) {

	rawXML := `
	<event name="sp_statement_completed" package="sqlserver" timestamp="2018-04-14T14:11:11.557Z">
		<data name="source_database_id"><value>12</value></data>
		<data name="object_id"><value>722361888</value></data>
		<data name="object_type"><value>8272</value><text><![CDATA[PROC]]></text></data>
		<data name="duration"><value>1759861</value></data>
		<data name="cpu_time"><value>481000</value></data>
		<data name="physical_reads"><value>24</value></data>
		<data name="logical_reads"><value>200444</value>
		</data><data name="writes"><value>66</value></data>
		<data name="row_count"><value>21766</value></data>
		<data name="last_row_count"><value>1</value></data>
		<data name="nest_level"><value>1</value></data>
		<data name="line_number"><value>152</value></data>
		<data name="offset"><value>9014</value></data>
		<data name="offset_end"><value>9122</value></data>
		<data name="object_name"><value><![CDATA[billing_AdjustBalances]]></value></data>
		<data name="statement"><value><![CDATA[exec billing_AccountBalance @AccountID]]></value></data>
		
		<action name="sql_text" package="sqlserver"><value><![CDATA[EXEC [dbo].[billing_AdjustBalances]      @ExecuteSQL = 1,   @ProcessAccounts = 2]]></value></action>
		<action name="session_id" package="sqlserver"><value>119</value></action>
		<action name="server_principal_name" package="sqlserver"><value><![CDATA[PROD\batchsvc]]></value></action>
		<action name="server_instance_name" package="sqlserver"><value><![CDATA[KCEUS-SRV1]]></value></action>
		<action name="query_hash" package="sqlserver"><value>0</value></action>
		<action name="database_name" package="sqlserver"><value><![CDATA[BILLING]]></value></action>
		<action name="client_pid" package="sqlserver"><value>17992</value></action>
		<action name="client_hostname" package="sqlserver"><value><![CDATA[KCE-ABAGENT01P]]></value></action>
		<action name="client_app_name" package="sqlserver"><value><![CDATA[.Net SqlClient Data Provider]]></value></action>
		<action name="collect_system_time" package="package0"><value>2018-04-14T14:11:11.558Z</value></action></event>
	`

	event, err := Parse(&i, rawXML)
	if err != nil {
		t.Error(err)
	}
	//t.Log("Raw Event: ", event)
	jsonBytes, err := json.Marshal(event)
	if err != nil {
		t.Error(err)
	}
	// t.Log(jsonBytes)
	jsonString := string(jsonBytes)
	t.Log("JSON String: ", jsonString)
}

func TestRPC(t *testing.T) {

	rawXML := `
	<event name="rpc_completed" package="sqlserver" timestamp="2018-04-14T16:33:06.973Z">
		<data name="cpu_time"><value>0</value></data>
		<data name="duration"><value>1070259</value></data>
		<data name="physical_reads"><value>0</value></data>
		<data name="logical_reads"><value>209</value></data>
		
		<data name="writes"><value>0</value></data>
		<data name="result"><value>0</value><text><![CDATA[OK]]></text></data>
		<data name="row_count"><value>7</value></data>
		<data name="connection_reset_option"><value>0</value><text><![CDATA[None]]></text></data>
		<data name="object_name"><value><![CDATA[sp_prepexec]]></value></data>
		<data name="statement"><value><![CDATA[declare @p1 int  set @p1=13  exec sp_prepexec @p1 output,NULL,N'    ;WITH CTE AS (     select      bus.server_name     ,bus.database_name     ,bus.backup_start_date     ,bus.type     ,ROW_NUMBER() OVER(PARTITION BY database_name, type ORDER BY backup_start_date DESC) AS RowNumber     ,bmf.physical_device_name    --,*     from msdb.dbo.backupset bus    join msdb.dbo.backupmediafamily bmf on bmf.media_set_id = bus.media_set_id   )   SELECT COALESCE(ag.name, server_name) as host,     @@SERVERNAME AS instance      ,database_name, backup_start_date, type, physical_device_name   FROM CTE   JOIN sys.databases d ON d.[name] = CTE.database_name   LEFT JOIN sys.availability_replicas r ON r.replica_id = d.replica_id   LEFT JOIN sys.availability_groups ag on ag.group_id = r.group_id   WHERE RowNumber = 1   order by database_name, backup_start_date desc       '  select @p1]]></value></data><data name="data_stream"><value></value></data>
		<data name="output_parameters"><value></value></data>
		
		<action name="sql_text" package="sqlserver"><value><![CDATA[    ;WITH CTE AS (     select      bus.server_name     ,bus.database_name     ,bus.backup_start_date     ,bus.type     ,ROW_NUMBER() OVER(PARTITION BY database_name, type ORDER BY backup_start_date DESC) AS RowNumber     ,bmf.physical_device_name    --,*     from msdb.dbo.backupset bus    join msdb.dbo.backupmediafamily bmf on bmf.media_set_id = bus.media_set_id   )   SELECT COALESCE(ag.name, server_name) as host,     @@SERVERNAME AS instance      ,database_name, backup_start_date, type, physical_device_name   FROM CTE   JOIN sys.databases d ON d.[name] = CTE.database_name   LEFT JOIN sys.availability_replicas r ON r.replica_id = d.replica_id   LEFT JOIN sys.availability_groups ag on ag.group_id = r.group_id   WHERE RowNumber = 1   order by database_name, backup_start_date desc       ]]></value></action>
		
		<action name="session_id" package="sqlserver"><value>70</value></action>
		<action name="server_principal_name" package="sqlserver"><value><![CDATA[PROD\sqlmonitor]]></value>
		</action>
		<action name="server_instance_name" package="sqlserver"><value><![CDATA[KCEUS-SRV1]]></value></action>
		<action name="query_hash" package="sqlserver"><value>0</value></action>
		<action name="database_name" package="sqlserver"><value><![CDATA[master]]></value></action>
		<action name="client_pid" package="sqlserver"><value>1616</value></action>
		<action name="client_hostname" package="sqlserver"><value><![CDATA[KCE-SQLMON03P]]></value></action>
		<action name="client_app_name" package="sqlserver"><value><![CDATA[IsItSQL]]></value></action>
		<action name="collect_system_time" package="package0"><value>2018-04-14T16:33:06.974Z</value></action>
	</event>
	`

	event, err := Parse(&i, rawXML)
	if err != nil {
		t.Error(err)
	}
	//t.Log("Raw Event: ", event)
	jsonBytes, err := json.Marshal(event)
	if err != nil {
		t.Error(err)
	}
	// t.Log(jsonBytes)
	jsonString := string(jsonBytes)
	t.Log("JSON String: ", jsonString)
}

func TestBatchStatement(t *testing.T) {

	rawXML := `

	<event name="sql_statement_completed" package="sqlserver" timestamp="2018-04-14T16:33:39.593Z">
	<data name="duration"><value>2768320</value></data>
	<data name="cpu_time"><value>2047000</value></data>
	<data name="physical_reads"><value>24518</value></data>
	<data name="logical_reads"><value>1153697</value></data>
	<data name="writes"><value>10371</value></data>
	<data name="row_count"><value>340590</value></data>
	<data name="last_row_count"><value>1</value></data>
	<data name="line_number"><value>1</value></data>
	<data name="offset"><value>0</value></data>
	<data name="offset_end"><value>58</value></data>
	<data name="statement"><value><![CDATA[EXECUTE billing_RefreshTransaction]]></value></data>
	<data name="parameterized_plan_handle"><value></value></data>
	
	<action name="sql_text" package="sqlserver"><value><![CDATA[EXECUTE billing_RefreshTransaction]]></value></action>
	<action name="session_id" package="sqlserver"><value>174</value></action>
	<action name="server_principal_name" package="sqlserver"><value><![CDATA[PROD\batchsvc]]></value></action>
	<action name="server_instance_name" package="sqlserver"><value><![CDATA[KCEUS-SRV1]]></value></action>
	<action name="query_hash" package="sqlserver"><value>0</value></action>
	<action name="database_name" package="sqlserver"><value><![CDATA[BILLING]]></value></action>
	<action name="client_pid" package="sqlserver"><value>6664</value></action>
	<action name="client_hostname" package="sqlserver"><value><![CDATA[LAE-BATCH]]></value></action>
	<action name="client_app_name" package="sqlserver"><value><![CDATA[.Net SqlClient Data Provider]]></value></action>
	<action name="collect_system_time" package="package0"><value>2018-04-14T16:33:39.593Z</value></action></event>

	`

	event, err := Parse(&i, rawXML)
	if err != nil {
		t.Error(err)
	}
	//t.Log("Raw Event: ", event)
	jsonBytes, err := json.Marshal(event)
	if err != nil {
		t.Error(err)
	}
	// t.Log(jsonBytes)
	jsonString := string(jsonBytes)
	t.Log("JSON String: ", jsonString)
}

func TestBatchCompleted(t *testing.T) {

	rawXML := `
	<event name="sql_batch_completed" package="sqlserver" timestamp="2018-04-14T16:33:39.593Z">
	<data name="cpu_time"><value>2047000</value></data>
	<data name="duration"><value>2768452</value></data>
	<data name="physical_reads"><value>24518</value></data>
	<data name="logical_reads"><value>1153697</value></data>
	<data name="writes"><value>10371</value></data>
	<data name="row_count"><value>340607</value></data>
	<data name="result"><value>0</value><text><![CDATA[OK]]></text></data>
	<data name="batch_text"><value><![CDATA[EXECUTE billing_RefreshTransaction]]></value></data>
	
	<action name="sql_text" package="sqlserver"><value><![CDATA[EXECUTE billing_RefreshTransaction]]></value></action>
	<action name="session_id" package="sqlserver"><value>174</value></action>
	<action name="server_principal_name" package="sqlserver"><value><![CDATA[PROD\batchsvc]]></value></action>
	<action name="server_instance_name" package="sqlserver"><value><![CDATA[KCEUS-SRV1]]></value></action>
	<action name="query_hash" package="sqlserver"><value>0</value></action>
	<action name="database_name" package="sqlserver"><value><![CDATA[BILLING]]></value></action>
	<action name="client_pid" package="sqlserver"><value>6664</value></action>
	<action name="client_hostname" package="sqlserver"><value><![CDATA[LAE-BATCH]]></value></action>
	<action name="client_app_name" package="sqlserver"><value><![CDATA[.Net SqlClient Data Provider]]></value></action>
	<action name="collect_system_time" package="package0"><value>2018-04-14T16:33:39.593Z</value></action></event>
	`

	event, err := Parse(&i, rawXML)
	if err != nil {
		t.Error(err)
	}
	//t.Log("Raw Event: ", event)
	jsonBytes, err := json.Marshal(event)
	if err != nil {
		t.Error(err)
	}
	// t.Log(jsonBytes)
	jsonString := string(jsonBytes)
	t.Log("JSON String: ", jsonString)
}

func TestDeadlockChain(t *testing.T) {

	rawXML := `
		<event name="lock_deadlock_chain" package="sqlserver" timestamp="2018-04-14T19:09:07.134Z">
			<data name="resource_type"><value>0</value><text><![CDATA[UNKNOWN_LOCK_RESOURCE]]></text>
			</data>
			<data name="mode"><value>0</value><text><![CDATA[NL]]></text></data>
			<data name="owner_type"><value>0</value><text><![CDATA[invalid]]></text></data>
			<data name="transaction_id"><value>0</value></data>
			<data name="database_id"><value>0</value></data>
			<data name="lockspace_workspace_id"><value>0x0000000000000000</value></data>
			<data name="lockspace_sub_id"><value>0</value></data>
			<data name="lockspace_nest_id"><value>0</value></data>
			<data name="resource_0"><value>0</value></data>
			<data name="resource_1"><value>0</value></data>
			<data name="resource_2"><value>0</value></data>
			<data name="deadlock_id"><value>10619177</value></data>
			<data name="object_id"><value>0</value></data>
			<data name="associated_object_id"><value>0</value></data>
			<data name="session_id"><value>0</value></data>
			<data name="resource_owner_type"><value>0x00000004</value><text><![CDATA[EXCHANGE]]></text></data>
			<data name="resource_description"><value><![CDATA[Parallel query worker thread was involved in a deadlock]]></value></data>
			<data name="database_name"><value><![CDATA[]]></value></data>
			<action name="session_id" package="sqlserver"><value>6</value></action>
			<action name="server_principal_name" package="sqlserver"><value><![CDATA[sa]]></value></action>
			<action name="server_instance_name" package="sqlserver"><value><![CDATA[KCEUS-SRV1]]></value></action>
			<action name="query_plan_hash" package="sqlserver"><value>0</value></action>
			<action name="query_hash" package="sqlserver"><value>0</value></action>
			<action name="plan_handle" package="sqlserver"><value>0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000</value></action>
			<action name="client_pid" package="sqlserver"><value>0</value></action>
			<action name="client_hostname" package="sqlserver"><value><![CDATA[]]></value></action>
			<action name="client_app_name" package="sqlserver"><value><![CDATA[]]></value></action>
			<action name="collect_system_time" package="package0"><value>2018-04-14T19:09:07.135Z</value></action>
		</event>
	`

	event, err := Parse(&i, rawXML)
	if err != nil {
		t.Error(err)
	}
	//t.Log("Raw Event: ", event)
	jsonBytes, err := json.Marshal(event)
	if err != nil {
		t.Error(err)
	}
	// t.Log(jsonBytes)
	jsonString := string(jsonBytes)
	t.Log("JSON String: ", jsonString)
}

func TestBlockedProcess(t *testing.T) {

	rawXML := `
	<event name="blocked_process_report" package="sqlserver" timestamp="2018-04-14T04:46:56.248Z">
		<data name="duration"><value>16912000</value></data>
		<data name="database_id"><value>7</value></data>
		<data name="object_id"><value>663673412</value></data>
		<data name="index_id"><value>0</value></data>
		<data name="lock_mode"><value>8</value><text><![CDATA[IX]]></text></data>
		<data name="transaction_id"><value>13574299574</value></data>
		<data name="resource_owner_type"><value>0x00000001</value><text><![CDATA[LOCK]]></text></data>
		<data name="blocked_process">
		<value>
			<blocked-process-report monitorLoop="2767730">   <blocked-process>    
			<process id="processa0ff06f088" taskpriority="0" logused="1296" waitresource="OBJECT: 7:663673412:0 " waittime="16912" 
				ownerId="13574299574" transactionname="INSERT" lasttranstarted="2018-04-13T23:46:39.333" 
				XDES="0x9657c40728" lockMode="IX" schedulerid="11" kpid="7864" status="suspended" spid="425" 
				sbid="0" ecid="0" priority="0" trancount="2" lastbatchstarted="2018-04-13T23:46:39.333" 
				lastbatchcompleted="2018-04-13T23:46:39.297" lastattention="1900-01-01T00:00:00.297" 
				clientapp="TestSVC_US" hostname="LAE-SRV4" hostpid="3708" loginname="SVCLogin" 
				isolationlevel="read committed (2)" xactid="13574299574" currentdb="7" lockTimeout="25000" 
				clientoption1="673316896" clientoption2="119840">     <executionStack>      
				<frame line="12" stmtstart="342" stmtend="998" 
				sqlhandle="0x03000700f35ec04c08a75b00d4a6000000000000000000000000000000000000000000000000000000000000"/>      
				<frame line="1" stmtstart="1102" stmtend="2420" 
				sqlhandle="0x020000009bafa4235a50665c45fc27ab0a885e60b17a09960000000000000000000000000000000000000000"/>      
				<frame line="1" sqlhandle="0x0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"/>     
				</executionStack>     
				<inputbuf>  (@P1 smallint,)insert into TXNDB.dbo.TXNTable (Attempt )    </inputbuf>    
				</process>   </blocked-process>   <blocking-process>    
				<process status="running" spid="315" sbid="0" ecid="0" priority="0" trancount="2" 
				lastbatchstarted="2018-04-13T23:45:35.547" lastbatchcompleted="2018-04-13T23:45:35.543" 
				lastattention="1900-01-01T00:00:00.543" clientapp=".Net SqlClient Data Provider" 
				hostname="KCE-BATCH" hostpid="15356" loginname="PROD\batchsvc" isolationlevel="read committed (2)" 
				xactid="13574282756" currentdb="7" lockTimeout="4294967295" clientoption1="673185824" clientoption2="128056">     
				<executionStack>      <frame line="6" stmtstart="178" stmtend="300" 
				sqlhandle="0x03000700f579e80a1f79e6001496000000000000000000000000000000000000000000000000000000000000"/>      
				<frame line="32" stmtstart="1146" stmtend="1248" 
					sqlhandle="0x03000700c2f9733d7e8838001c9e000000000000000000000000000000000000000000000000000000000000"/>      
					<frame line="48" stmtstart="3758" stmtend="4610" 
					sqlhandle="0x03000700d8532133728838001c9e000001000000000000000000000000000000000000000000000000000000"/>      
					<frame line="30" stmtstart="2918" stmtend="2986" 
					sqlhandle="0x03000700311b09025447ff001e9e000001000000000000000000000000000000000000000000000000000000"/>      
			<frame line="1" stmtend="46" 
				sqlhandle="0x01000700e4710322f09cc4f9d500000000000000000000000000000000000000000000000000000000000000"/>    
				 </executionStack>     
				 <inputbuf>  EXECUTE psx_NightlyTasks   </inputbuf>    </process>   </blocking-process>  </blocked-process-report>  
		</value></data>
		<data name="database_name"><value><![CDATA[TXNDB]]></value></data>
		
		<action name="session_id" package="sqlserver"><value>4</value></action>
		<action name="server_principal_name" package="sqlserver"><value><![CDATA[sa]]></value></action>
		<action name="server_instance_name" package="sqlserver"><value><![CDATA[KCEUS-DBTXN01P]]></value></action>
		<action name="query_hash" package="sqlserver"><value>0</value></action>
		<action name="client_pid" package="sqlserver"><value>0</value></action>
		<action name="client_hostname" package="sqlserver"><value><![CDATA[]]></value></action>
		<action name="client_app_name" package="sqlserver"><value><![CDATA[]]></value></action>
		<action name="collect_system_time" package="package0"><value>2018-04-14T04:46:56.249Z</value></action>
	</event>
	`

	event, err := Parse(&i, rawXML)
	if err != nil {
		t.Error(err)
	}
	//t.Log("Raw Event: ", event)
	jsonBytes, err := json.Marshal(event)
	if err != nil {
		t.Error(err)
	}
	// t.Log(jsonBytes)
	jsonString := string(jsonBytes)
	t.Log("JSON String: ", jsonString)
}
