package config

import (
	"testing"
	"time"

	"github.com/billgraziano/toml"
	"github.com/stretchr/testify/assert"
)

func TestStartAt(t *testing.T) {
	assert := assert.New(t)
	var c = `
	start_at = "2018-01-01T13:14:15Z"
	`
	src := Source{}
	_, err := toml.Decode(c, &src)
	if err != nil {
		t.Errorf("decode err = %v", err)
	}
	dt, err := time.Parse(time.RFC3339, "2018-01-01T13:14:15Z")
	if err != nil {
		t.Errorf("parse err = %v", err)
	}
	assert.Equal(dt, src.StartAt, "expected: %v; got: %v", dt, src.StartAt)
}

func TestLookBack(t *testing.T) {
	assert := assert.New(t)
	var cfg = `
	look_back = "1h"
	`
	src := Source{}
	_, err := toml.Decode(cfg, &src)
	assert.NoError(err, "err = %v", err)
}

func TestSourceBackConfig(t *testing.T) {
	assert := assert.New(t)
	dt, err := time.Parse(time.RFC3339, "2018-01-01T13:14:15Z")
	assert.NoError(err)
	src := Source{
		StartAt:     dt,
		LookBackRaw: "1h",
	}
	err = src.processLookback()
	assert.NoError(err, "processlookback err = %v", err)
	assert.WithinDuration(time.Now(), src.StartAt, 2*time.Hour)
}

func TestConfigDefaultLookbackConfig(t *testing.T) {
	assert := assert.New(t)
	var err error
	// dt, err := time.Parse(time.RFC3339, "2018-01-01T13:14:15Z")
	// assert.NoError(err)
	cfg := Config{
		Defaults: Source{
			LookBackRaw: "100h",
		},
		Sources: []Source{
			{
				FQDN:        "test",
				LookBackRaw: "1h",
			},
		},
	}
	err = cfg.Defaults.processLookback()
	assert.NoError(err)
	err = cfg.Sources[0].processLookback()
	assert.NoError(err)
	assert.WithinDuration(time.Now(), cfg.Sources[0].StartAt, 2*time.Hour)
}
