package prom

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrometheusLabel(t *testing.T) {
	assert := assert.New(t)
	type test struct {
		domain string
		server string
		want   string
	}
	tests := []test{
		{domain: "TeST", server: "JUNK", want: "test_junk"},
		{domain: "TeST", server: "D40\\PROD", want: "test_d40_prod"},
		{server: "D40\\PROD", want: "d40_prod"},
		{server: "D40", want: "d40"},
	}
	for _, tc := range tests {
		got := ServerLabel(tc.domain, tc.server)
		assert.Equal(tc.want, got)
	}
}
