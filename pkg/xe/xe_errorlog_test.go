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
		//Need to figure out the language and work backwords to extract the text
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
	}

	for _, tc := range tt {
		e := Event{}
		e.Set("message", tc.raw)
		e.parseErrorLogMessage()
		assert.Equal(tc.raw, e.GetString("errorlog_raw"))
		assert.Equal(tc.proc, e.GetString("errorlog_process"))
		assert.Equal(tc.msg, e.GetString("errorlog_message"))
		assert.Equal(tc.client, e.GetString("errorlog_client"))
		if tc.err != 0 {
			num, ok := e.GetInt64("error_number")
			assert.True(ok)
			assert.Equal(tc.err, num)
			sev, ok := e.GetInt64("severity")
			assert.True(ok)
			assert.Equal(tc.sev, sev)
			state, ok := e.GetInt64("state")
			assert.True(ok)
			assert.Equal(tc.state, state)
		}
	}
}
