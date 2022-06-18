package logstash

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestUpperCaseArray(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	src := `
		{
			"mssql": {
				"name": "error_reported",
				"error_number": 2627,
				"xe_session_name": "logstash_events",
				"mssql_ag_listener": [
					"SQLPDDSI",
					"sqlpdjde",
					"SQLPDJWS"
				]
			}
		}
`
	result := gjson.Get(src, "mssql.mssql_ag_listener")
	require.True(result.Exists())
	newJson, err := ProcessUpperLower(src, []string{"mssql.name", "mssql.mssql_ag_listener"}, []string{})
	assert.NoError(err)
	name := gjson.Get(newJson, "mssql.name")
	require.Equal(gjson.String, name.Type)
	assert.Equal("ERROR_REPORTED", name.String())
	ag := gjson.Get(newJson, "mssql.mssql_ag_listener.1")
	assert.True(ag.Exists())
	assert.Equal("SQLPDJDE", ag.String())
}
