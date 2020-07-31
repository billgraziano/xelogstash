package xe

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseErrorLog(t *testing.T) {
	assert := assert.New(t)
	type test struct {
		raw  string
		dt   string
		tm   string
		proc string
		msg  string
	}
	tt := []test{
		{
			raw:  "2020-07-12 14:57:35.67 spid26s     The Database Mirroring endpoint is in disabled or stopped state.",
			dt:   "2020-07-12",
			tm:   "14:57:35.67",
			proc: "spid26s",
			msg:  "The Database Mirroring endpoint is in disabled or stopped state.",
		},
		{
			raw:  "2020-07-12 14:57:35.47 Server      SQL Server is attempting to register a Service Principal Name (SPN)...",
			dt:   "2020-07-12",
			tm:   "14:57:35.47",
			proc: "server",
			msg:  "SQL Server is attempting to register a Service Principal Name (SPN)...",
		},
		{
			raw:  "2020-07-12 14:52:39.57 Backup      BACKUP DATABASE successfully processed 27154 pages in 0.207 seconds (1024.833 MB/sec).  ",
			dt:   "2020-07-12",
			tm:   "14:52:39.57",
			proc: "backup",
			msg:  "BACKUP DATABASE successfully processed 27154 pages in 0.207 seconds (1024.833 MB/sec).",
		},
		{
			raw:  "2020-07-12 15:29:10.11 Logon       Error: 18456, Severity: 14, State: 5.  2020-07-12 15:29:10.11 Logon       Login failed for user 'asdfasfd'. Reason: Could not find a login matching the name provided. [CLIENT: 192.168.7.40]  ",
			dt:   "2020-07-12",
			tm:   "15:29:10.11",
			proc: "logon",
			msg:  "Error: 18456, Severity: 14, State: 5. Login failed for user 'asdfasfd'. Reason: Could not find a login matching the name provided. [CLIENT: 192.168.7.40]",
		},
		{
			raw:  "2020-07-12 15:29:10.11 Logon       Error: 18456, Severity: 14, State: 5.  2020-07-12 15:29:10.11 Logon       Login",
			dt:   "2020-07-12",
			tm:   "15:29:10.11",
			proc: "logon",
			msg:  "Error: 18456, Severity: 14, State: 5. Login",
		},
		{
			raw:  "2020-07-12 15:29:10.11 Logon       Error: 18456, Severity: 14, State: 5.  2020-07-12 15:29:10.11 Logon",
			dt:   "2020-07-12",
			tm:   "15:29:10.11",
			proc: "logon",
			msg:  "",
		},
	}
	for _, tc := range tt {
		e := Event{}
		e.Set("message", tc.raw)
		e.parseErrorLogMessage()
		assert.Equal(tc.dt, e.GetString("errorlog.date"))
		assert.Equal(tc.tm, e.GetString("errorlog.time"))
		assert.Equal(tc.proc, e.GetString("errorlog.process"))
		assert.Equal(tc.msg, e.GetString("errorlog.message"))
	}
}
