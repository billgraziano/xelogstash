package xe

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var domainLoginEvent = `
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

var sqlLoginEvent = `
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
	<action name="server_principal_name" package="sqlserver"><value><![CDATA[testuser]]></value></action>
</event>
`

func TestParsingSQLLogin(t *testing.T) {
	info := sqlinfo()
	assert := assert.New(t)
	require := require.New(t)
	event, err := Parse(&info, sqlLoginEvent, false)
	if err != nil {
		t.Error(err)
	}
	require.Equal("login", event["name"])
	assert.False(event.Exists("xe_login_domain"))
	assert.False(event.Exists("xe_login_domain_user"))
	assert.Equal("testuser", event["server_principal_name"])
}

func TestParsingDomainLogin(t *testing.T) {
	info := sqlinfo()
	assert := assert.New(t)
	require := require.New(t)
	event, err := Parse(&info, domainLoginEvent, false)
	if err != nil {
		t.Error(err)
	}
	require.Equal("login", event["name"])
	require.True(event.Exists("xe_login_domain"), "xe_login_domain is missing")
	assert.Equal("D30", event["xe_login_domain"])
	require.True(event.Exists("xe_login_domain_user"))
	assert.Equal("Bill", event["xe_login_domain_user"])
	assert.Equal(`D30\Bill`, event["server_principal_name"])
}

func sqlinfo() SQLInfo {
	return SQLInfo{
		Server:         "D30",
		Domain:         "WORKGROUP",
		Computer:       "D30",
		ProductLevel:   "Test",
		ProductRelease: "Test",
		Version:        "13.0",
		Actions: map[string]string{
			"plan_handle":            "binary_data",
			"query_hash_signed":      "int64",
			"query_plan_hash_signed": "int64",
			"query_plan_hash":        "uint64",
			"query_hash":             "uint64",
		},
		Fields: map[FieldTypeKey]string{
			{"error_reported", "error_number"}: "int32",
			{"error_reported", "state"}:        "int32",
		},
	}
}
