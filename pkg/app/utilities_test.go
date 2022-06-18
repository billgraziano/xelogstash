package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDuration(t *testing.T) {
	assert := assert.New(t)
	type test struct {
		src  string
		want string
	}

	tests := []test{
		{"25h3m", "1d1h"},
		{"50h59m", "2d2h"},
		{"23h59m59s", "23h59m"},
		{"1h", "1h0m"},
		{"59m", "59m0s"},
		{"67.25s", "1m7s"},
	}

	for _, tc := range tests {
		dur, err := time.ParseDuration(tc.src)
		assert.NoError(err)
		got := fmtduration(dur)
		assert.Equal(tc.want, got)
	}
}
