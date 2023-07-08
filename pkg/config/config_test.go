package config

import (
	"os"
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

func TestSetFromEnv(t *testing.T) {
	assert := assert.New(t)
	var str string
	var err error

	// Just a value
	str, err = setFromEnv("user")
	assert.NoError(err)
	assert.Equal("user", str)

	// Match, but empty variable
	str, err = setFromEnv("$(env:)")
	assert.Error(err)
	assert.Equal("$(env:)", str)

	// Match, but variable isn't set
	str, err = setFromEnv("$(env:SQLXE_USER)")
	assert.Error(err)
	assert.Equal("$(env:SQLXE_USER)", str)

	// Match and variable is set
	os.Setenv("SQLXE_USER", "myuser")
	str, err = setFromEnv("$(env:SQLXE_USER)")
	assert.NoError(err)
	assert.Equal("myuser", str)

	// Sketchy casing.  This is different in Windows and Linux :(
	// str, err = setFromEnv("$(ENV:SQLXE_user)")
	// assert.NoError(err)
	// assert.Equal("myuser", str)
}

func TestProcessEnvVariables(t *testing.T) {
	assert := assert.New(t)
	os.Setenv("SQLXE_UP", "userpass")
	var cfg Config
	cfg.Defaults.User = "$(env:SQLXE_UP)"
	cfg.Defaults.Password = "$(env:SQLXE_UP)"
	cfg.Elastic.Username = "$(env:SQLXE_UP)"
	cfg.Elastic.Password = "$(env:SQLXE_UP)"
	cfg.Sources = []Source{{User: "$(env:SQLXE_UP)", Password: "$(env:SQLXE_UP)"}}
	err := cfg.processEnvVariables()
	assert.NoError(err)
	assert.Equal("userpass", cfg.Defaults.User)
	assert.Equal("userpass", cfg.Defaults.Password)
	assert.Equal("userpass", cfg.Elastic.Username)
	assert.Equal("userpass", cfg.Elastic.Password)
	assert.Equal("userpass", cfg.Sources[0].User)
	assert.Equal("userpass", cfg.Sources[0].Password)
}
