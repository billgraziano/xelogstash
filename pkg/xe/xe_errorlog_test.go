package xe

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseErrorLog(t *testing.T) {
	assert := assert.New(t)
	type test struct {
		raw    string
		proc   string
		msg    string
		err    int64
		sev    int64
		state  int64
		client string
	}
	tt := []test{
		{
			raw:  "2020-07-12 14:57:35.67 spid26s     The Database Mirroring endpoint is in disabled or stopped state.",
			proc: "spid26s",
			msg:  "The Database Mirroring endpoint is in disabled or stopped state.",
		},
		{
			raw:  "2020-07-12 14:57:35.47 Server      SQL Server is attempting to register a Service Principal Name (SPN)...",
			proc: "server",
			msg:  "SQL Server is attempting to register a Service Principal Name (SPN)...",
		},
		{
			raw:  "2020-07-12 14:52:39.57 Backup      BACKUP DATABASE successfully processed 27154 pages in 0.207 seconds (1024.833 MB/sec).  ",
			proc: "backup",
			msg:  "BACKUP DATABASE successfully processed 27154 pages in 0.207 seconds (1024.833 MB/sec).",
		},
		{
			raw:    "2020-07-12 15:29:10.11 Logon       Error: 18456, Severity: 14, State: 5.  2020-07-12 15:29:10.11 Logon       Login failed for user 'asdfasfd'. Reason: Could not find a login matching the name provided. [CLIENT: 192.168.7.40]  ",
			proc:   "logon",
			msg:    "Error: 18456, Severity: 14, State: 5.  Login failed for user 'asdfasfd'. Reason: Could not find a login matching the name provided. [CLIENT: 192.168.7.40]",
			err:    18456,
			sev:    14,
			state:  5,
			client: "192.168.7.40",
		},
		{
			raw:   "2020-07-12 15:29:10.11 Logon       Error: 18456, Severity: 14, State: 5.  2020-07-12 15:29:10.11 Logon       Login",
			proc:  "logon",
			msg:   "Error: 18456, Severity: 14, State: 5.  Login",
			err:   18456,
			sev:   14,
			state: 5,
		},
		{
			// This originally failed.  But I think we should set what we can here
			raw:   "2020-07-12 15:29:10.11 Logon       Error: 18456, Severity: 99, State: 5.  2020-07-12 15:29:10.11 Logon",
			proc:  "logon",
			msg:   "Error: 18456, Severity: 99, State: 5. ",
			err:   18456,
			sev:   99,
			state: 5,
		},
		//Test is broken
		//Need to figure out the language and work backwards to extract the text
		{
			raw:    "2020-08-06 07:28:24.76 Logon       Login succeeded for user 'D40\\graz'. Connection made using Windows authentication. [CLIENT: <local machine>]  ",
			proc:   "logon",
			msg:    "Login succeeded for user 'D40\\graz'. Connection made using Windows authentication. [CLIENT: <local machine>]",
			client: "<local machine>",
		},
		{
			raw:    "2024-07-19 07:39:20.95 Logon       Error: 18456, Severity: 14, State: 5.  2024-07-19 07:39:20.95 Logon       Login failed for user 'hjkhkj'. Reason: Could not find a login matching the name provided. [CLIENT: <local machine>]",
			proc:   "logon",
			client: "<local machine>",
			msg:    "Error: 18456, Severity: 14, State: 5.  Login failed for user 'hjkhkj'. Reason: Could not find a login matching the name provided. [CLIENT: <local machine>]",
		},
		{
			raw:    "2024-07-19 07:39:20.95 Logon       Error: 18456, Severity: 14, State: 5.  2024-07-19 07:39:20.95 Logon       Login failed for user 'hjkhkj'. Reason: Could not find a login matching the name provided. [CLIENT: 10.10.32.1]",
			proc:   "logon",
			client: "10.10.32.1",
			msg:    "Error: 18456, Severity: 14, State: 5.  Login failed for user 'hjkhkj'. Reason: Could not find a login matching the name provided. [CLIENT: 10.10.32.1]",
		},
		{
			raw:    "2024-07-19 07:39:20.95 Logon       Error: 18456, Severity: 14, State: 5.  2024-07-19 07:39:20.95 Logon       Login failed for user 'hjkhkj'. Reason: Could not find a login matching the name provided. [CLIENT: ::1]",
			proc:   "logon",
			client: "::1",
			msg:    "Error: 18456, Severity: 14, State: 5.  Login failed for user 'hjkhkj'. Reason: Could not find a login matching the name provided. [CLIENT: ::1]",
		},
		{
			raw:  "2026-08-14 22:03:19.55 spid174     I/O was resumed on database TXNDB. No user action is required.",
			proc: "info",
			msg:  "I/O was resumed on database TXNDB. No user action is required.",
		},
		{
			raw:  "2026-08-14 22:03:19.45 spid179     I/O is frozen on database master. No user action is required. However, if I/O is not resumed promptly, you could cancel the backup.",
			proc: "info",
		},
		{
			raw:  "2026-08-15 09:11:29.94 spid52      DBCC CHECKDB (TXNDB) WITH physical_only executed by NT SERVICE\\SQLSERVERAGENT found 0 errors and repaired 0 errors. Elapsed time: 0 hours 0 minutes 0 seconds.  Internal database snapshot has split point LSN = 00000a29:000004e6:0001 and first LSN = 00000a29:000004e4:0001.",
			proc: "dbcc",
		},
		{
			raw:  "2026-08-15 03:31:31.41 spid25s     [INFO] Database Id: [9],pru->IsReadOnly:false",
			proc: "info",
		},
		{
			raw:  "2026-08-15 02:16:07.54 spid104s    Error: 41145, Severity: 16, State: 1. 2026-08-15 02:16:07.54 spid104s    Cannot join database 'TXNDB' to availability group 'TXNAG'.  The database has already joined the availability group.  This is an informational message.  No user action is required.",
			proc: "info",
		}, {
			raw:  "2026-08-15 02:16:07.54 spid104s    Always On is doing stuff.",
			proc: "alwayson",
		},
	}

	for _, tc := range tt {
		e := Event{}
		e.Set("message", tc.raw)
		e.parseErrorLogMessage()
		assert.Equal(tc.raw, e.GetString("errorlog_raw"), tc.raw)
		assert.Equal(tc.proc, e.GetString("errorlog_process"), tc.raw)
		if tc.msg != "" {
			assert.Equal(tc.msg, e.GetString("errorlog_message"), tc.raw)
		}
		assert.Equal(tc.client, e.GetString("xe_client_address"), tc.raw)
		if tc.err != 0 {
			num, ok := e.GetInt64("error_number")
			assert.True(ok, tc.raw)
			assert.Equal(tc.err, num, tc.raw)
			sev, ok := e.GetInt64("severity")
			assert.True(ok, tc.raw)
			assert.Equal(tc.sev, sev, tc.raw)
			state, ok := e.GetInt64("state")
			assert.True(ok, tc.raw)
			assert.Equal(tc.state, state, tc.raw)
		}
	}
}
