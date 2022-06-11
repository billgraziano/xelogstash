package xe

import (
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnits(t *testing.T) {
	i = SQLInfo{
		Server:         "D30",
		Domain:         "WORKGROUP",
		Computer:       "D30",
		ProductLevel:   "Test",
		ProductRelease: "Test",
		Version:        "13.0",
	}

	rawEvent := `
<event name="rpc_completed"	package="sqlserver" timestamp="2022-06-09T22:12:25.970Z">
	<data name="cpu_time"><value>8100123</value></data>
	<data name="duration"><value>123456789</value></data>
	<data name="physical_reads"><value>128</value></data>
	<data name="logical_reads"><value>896</value></data>
	<data name="writes"><value>0</value></data>
	
	<data name="statement"><value><![CDATA[SELECT 1]]></value></data>
	<action name="sql_text" package="sqlserver"><value><![CDATA[SELECT 2]]></value></action>
</event>

`
	assert := assert.New(t)
	event, err := Parse(&i, rawEvent)
	assert.NoError(err)
	assert.Equal(24, len(event))

	dur, ok := event.GetInt64("duration_sec")
	assert.True(ok)
	assert.Equal(int64(123), dur)

	pr, ok := event.GetInt64("physical_reads_mb")
	assert.True(ok)
	assert.Equal(int64(1), pr)

	lr, ok := event.GetInt64("logical_reads_mb")
	assert.True(ok)
	assert.Equal(int64(7), lr)

	wr, ok := event.GetInt64("writes_mb")
	assert.True(ok)
	assert.Equal(int64(0), wr)

	// Print the map if it fails
	keys := make([]string, 0, len(event))
	for k := range event {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		fmt.Printf("%25s:  value: %v\n", k, event[k])
	}
}
