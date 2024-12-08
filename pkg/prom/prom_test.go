package prom

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrometheusLabel(t *testing.T) {
	assert := assert.New(t)
	type test struct {
		server string
		want   string
	}
	tests := []test{
		{server: "D40\\PROD", want: "d40__prod"},
		{server: "D40", want: "d40"},
		{server: "", want: ""},
	}
	for _, tc := range tests {
		got := ServerLabel(tc.server)
		assert.Equal(tc.want, got)
	}
}
