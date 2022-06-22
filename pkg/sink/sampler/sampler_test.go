package sampler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestSampler(t *testing.T) {

	rawEvent := `
	{
		"source_a2": "BIG",
		"source_a1": "GO",
		"add_git_hash": "abcdef",
		"add_git_describe": 1.1,
		"add_version": "1.1",
		"@timestamp": "2022-06-02T22:53:21.759Z",
		"mssql": {
			"client_app_name": "sqlxewriter.exe",
			"client_hostname": "D40",
			"database_name": "master",
			"duration": 368,
			"mssql_computer": "first_name",
			"mssql_domain": "D2014",
			"mssql_product_version": "12.0.5223.6",
			"mssql_server_name": "first_name",
			"mssql_version": "SQL Server 2014 SP2",
			"name": "attention",
			"request_id": 0,
			"server_instance_name": "D40\\SQL2014",
			"server_principal_name": "D40\\graz",
			"sql_text": "SELECT object_name, event_data, file_name, file_offset FROM sys.fn_xe_file_target_read_file('system_health*.xel', NULL, 'C:\\Program Files\\Microsoft SQL Server\\MSSQL12.SQL2014\\MSSQL\\Log\\system_health_0_132984733185810000.xel', 346624);",
			"timestamp": "2022-06-02T22:53:21.759Z",
			"xe_acct_app": "D40\\graz - sqlxewriter.exe",
			"xe_acct_app_client": "D40\\graz - sqlxewriter.exe (D40)",
			"xe_category": "attention",
			"xe_description": "(D: 368μs) SELECT object_name, event_data, file_name, file_offset FROM sys.fn_xe_file_target_read_file('system_health*.xel', NULL, 'C:\\Program Files\\Microsoft SQL Server\\MSSQL12.SQL2014\\MSSQL\\Log\\system_health_0_132984733185810000.xel', 346624);",
			"xe_file_name": "C:\\Program Files\\Microsoft SQL Server\\MSSQL12.SQL2014\\MSSQL\\Log\\Attentions_0_132986837481060000.xel",
			"xe_file_offset": 11776,
			"xe_session_name": "Attentions",
			"xe_severity_keyword": "info",
			"xe_severity_value": 6
		}
	}
	`
	fileName := "./sinks/sampler/attention.json"

	assert := assert.New(t)
	logger := log.NewEntry(log.New())
	s := New(".", 1*time.Minute)
	s.logger = logger
	s.fs = afero.NewMemMapFs()
	mock := clock.NewMock()
	s.clock = mock
	ctx := context.Background()
	err := s.Open(ctx, "id")
	assert.NoError(err)
	_, err = s.Write(ctx, "attention", rawEvent)
	assert.NoError(err)
	exists, err := afero.Exists(s.fs, fileName)
	assert.NoError(err)
	assert.True(exists)
	err = s.fs.Chtimes(fileName, s.clock.Now(), s.clock.Now())
	assert.NoError(err)
	fi, err := s.fs.Stat(fileName)
	assert.NoError(err)
	fmt.Println(fi.Name(), fi.ModTime(), fi.Size())
	firstWrite := fi.ModTime()
	assert.Equal(firstWrite, s.clock.Now())

	// advance 30 seconds and don't write
	mock.Add(30 * time.Second)
	_, err = s.Write(ctx, "attention", rawEvent)
	assert.NoError(err)
	fi, err = s.fs.Stat(fileName)
	assert.NoError(err)
	assert.Equal(firstWrite, fi.ModTime()) // hasn't changed

	// advance 90 seconds and write
	mock.Add(90 * time.Second)
	_, err = s.Write(ctx, "attention", rawEvent)
	assert.NoError(err)
	fi, err = s.fs.Stat(fileName)
	assert.NoError(err)
	assert.NotEqual(firstWrite, fi.ModTime()) // has changed
	// t.Error("boom")

}
