package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

// Regex to match `$(env:VAR)` in user and password fields
var envRegex = regexp.MustCompile(`(?im)^\$\(env:(?P<var>\w*)\)$`)

// setFromEnv looks for a pattern like $(env:VAR).
// If found, it returns the value of that environment variable
// If that variable isn't set, it returns an error
func setFromEnv(s string) (string, error) {
	var variable string
	if strings.TrimSpace(s) == "" {
		return s, nil
	}
	// match the regex, return if no match
	match := envRegex.FindStringSubmatch(s)
	if len(match) == 0 {
		return s, nil
	}
	for i, matchName := range envRegex.SubexpNames() {
		if i != 0 && matchName != "" {
			variable = match[i]
		}
	}
	// if we didn't get a variable, this is a parsing error
	if variable == "" {
		return s, fmt.Errorf("missing variable: '%s'", s)
	}

	val := os.Getenv(variable)
	if val == "" {
		return s, fmt.Errorf("empty variable: '%s'", variable)
	}

	return val, nil
}

// processEnvVariables goes through a configuration file and optionally replaces all the
// user and password fields with the specified environment variables
func (cfg *Config) processEnvVariables() error {
	var err error
	cfg.Defaults.User, err = setFromEnv(cfg.Defaults.User)
	if err != nil {
		return errors.Wrap(err, "defaults.user")
	}
	cfg.Defaults.Password, err = setFromEnv(cfg.Defaults.Password)
	if err != nil {
		return errors.Wrap(err, "defaults.password")
	}
	for i := range cfg.Sources {
		cfg.Sources[i].User, err = setFromEnv(cfg.Sources[i].User)
		if err != nil {
			return errors.Wrapf(err, "sources[%s].user", cfg.Sources[i].FQDN)
		}
		cfg.Sources[i].Password, err = setFromEnv(cfg.Sources[i].Password)
		if err != nil {
			return errors.Wrapf(err, "sources[%s].password", cfg.Sources[i].FQDN)
		}
	}
	cfg.Elastic.Username, err = setFromEnv(cfg.Elastic.Username)
	if err != nil {
		return errors.Wrap(err, "elastic.username")
	}
	cfg.Elastic.Password, err = setFromEnv(cfg.Elastic.Password)
	if err != nil {
		return errors.Wrap(err, "elastic.password")
	}
	return nil
}
