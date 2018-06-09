package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Showmax/go-fqdn"

	"github.com/billgraziano/toml"
	"github.com/pkg/errors"
)

// JobsAll, JobsFailed, JobsNone are possible values for including agent jobs
const (
	JobsAll    = "all"
	JobsFailed = "failed"
	JobsNone   = "none"
)

// Source defines a source of extended event information
type Source struct {
	FQDN           string
	Sessions       []string
	IgnoreSessions bool `toml:"ignore_sessions"` // if true, skip XE sessions
	Prefix         string
	AgentJobs      string
	PayloadField   string `toml:"payload_field_name"`
	TimestampField string `toml:"timestamp_field_name"`
	Rows           int

	Adds           map[string]string
	Copies         map[string]string
	Moves          map[string]string
	ExcludedEvents []string //XE events that are excluded.  Mostly from system health

	RawAdds   []string `toml:"adds"`
	RawCopies []string `toml:"copies"`
	RawMoves  []string `toml:"moves"`
}

// App defines the application configuration
type App struct {
	Workers  int
	Logstash string
	Samples  bool // Print sample JSON to stdout
	Summary  bool // Print a summary to stdout
	// Enables a web server on :8080 with basic metrics
	HTTPMetrics bool `toml:"http_metrics"`
}

// AppLog controls the application logging
type AppLog struct {
	Logstash       string
	PayloadField   string `toml:"payload_field_name"`
	TimestampField string `toml:"timestamp_field_name"`
	Samples        bool

	Adds   map[string]string
	Copies map[string]string
	Moves  map[string]string

	RawAdds   []string `toml:"adds"`
	RawCopies []string `toml:"copies"`
	RawMoves  []string `toml:"moves"`
}

// Config defines the configuration read from the TOML file
type Config struct {
	App      App
	AppLog   AppLog
	Defaults Source   `toml:"defaults"`
	Sources  []Source `toml:"source"`
	MetaData toml.MetaData
}

// Get the configuration from a configuration file
func Get(f string, version string) (config Config, err error) {
	md, err := toml.DecodeFile(f, &config)
	if err != nil {
		return config, errors.Wrap(err, "toml.decode")
	}

	config.MetaData = md
	if config.AppLog.TimestampField == "" {
		return config, fmt.Errorf("applog.timestamp_field_name is required.  Suggest \"@timestamp\"")
	}

	err = config.decodekv(version)
	if err != nil {
		return config, errors.Wrap(err, "decodekv")
	}

	config.setDefaults()
	err = config.Defaults.validate()
	if err != nil {
		return config, errors.Wrap(err, "validate")
	}

	for _, s := range config.Sources {
		if s.FQDN == "" {
			return config, errors.New("source without fqdn")
		}
		err = s.validate()
		if err != nil {
			return config, errors.Wrap(err, "validate")
		}
	}

	return config, err
}

func (s *Source) validate() error {

	// if s.PayloadField != "" && s.TimestampField == "" {
	// 	return fmt.Errorf("%s - %s: if payload is set, timestamp must be set", s.Prefix, s.FQDN)
	// }

	// if s.PayloadField == "" && s.TimestampField != "" {
	// 	return fmt.Errorf("%s - %s: if timestamp is set, payload must be set", s.Prefix, s.FQDN)
	// }
	// if timestamp_field_name is provided, it must have a value
	if s.TimestampField == "" {
		return fmt.Errorf("timestamp_field_name must have a value.  suggest \"@timestamp\"")
	}

	if s.AgentJobs != JobsAll && s.AgentJobs != JobsFailed && s.AgentJobs != JobsNone && s.AgentJobs != "" {
		return fmt.Errorf("agentjobs must be all, none, or failed or not specified")
	}

	return nil
}

func (c *Config) setLowerCase() {
	// excluded events
	for i := range c.Defaults.ExcludedEvents {
		c.Defaults.ExcludedEvents[i] = strings.ToLower(c.Defaults.ExcludedEvents[i])
	}

	for j := range c.Sources {
		for i := range c.Sources[j].ExcludedEvents {
			c.Sources[j].ExcludedEvents[i] = strings.ToLower(c.Sources[j].ExcludedEvents[i])
		}
	}

}

func (c *Config) decodekv(version string) error {
	var err error
	c.Defaults.Adds, err = buildmap(c.Defaults.RawAdds, version)
	if err != nil {
		return errors.Wrap(err, "default-adds")
	}
	if c.Defaults.Copies, err = buildmap(c.Defaults.RawCopies, version); err != nil {
		return errors.Wrap(err, "default-copies")
	}
	if c.Defaults.Moves, err = buildmap(c.Defaults.RawMoves, version); err != nil {
		return errors.Wrap(err, "default-renames")
	}

	for i := range c.Sources {
		if c.Sources[i].Adds, err = buildmap(c.Sources[i].RawAdds, version); err != nil {
			return errors.Wrap(err, "source-adds")
		}
		if c.Sources[i].Copies, err = buildmap(c.Sources[i].RawCopies, version); err != nil {
			return errors.Wrap(err, "source-copies")
		}
		if c.Sources[i].Moves, err = buildmap(c.Sources[i].RawMoves, version); err != nil {
			return errors.Wrap(err, "source-renames")
		}
	}

	// Process the app settings
	if c.AppLog.Adds, err = buildmap(c.AppLog.RawAdds, version); err != nil {
		return errors.Wrap(err, "buildmap.adds")
	}

	if c.AppLog.Copies, err = buildmap(c.AppLog.RawCopies, version); err != nil {
		return errors.Wrap(err, "buildmap.copies")
	}

	if c.AppLog.Moves, err = buildmap(c.AppLog.RawMoves, version); err != nil {
		return errors.Wrap(err, "buildmap.moves")
	}

	// if c.App.Copies, err = buildmap(c.App.RawCopies); err != nil {
	// 	return errors.Wrap(err, "app-copies")
	// }

	// if c.App.Renames, err = buildmap(c.App.RawMoves); err != nil {
	// 	return errors.Wrap(err, "app-renames")
	// }

	return err
}

func buildmap(a []string, version string) (map[string]string, error) {
	m := make(map[string]string)
	var err error

	exeNamePath, err := os.Executable()
	if err != nil {
		return m, errors.Wrap(err, "os.executable")
	}
	exeName := filepath.Base(exeNamePath)
	exeNamePath = strings.Replace(exeNamePath, "\\", "\\\\", -1)

	for _, s := range a {
		kv := strings.Split(s, ":")
		if len(kv) != 2 {
			return m, fmt.Errorf("expected \"value:value\". got \"%s\"", s)
		}

		// process any substitutions
		value := kv[1]

		value = strings.Replace(value, "$(EXENAMEPATH)", exeNamePath, -1)
		value = strings.Replace(value, "$(EXENAME)", exeName, -1)
		value = strings.Replace(value, "$(PID)", strconv.Itoa(os.Getpid()), -1)
		value = strings.Replace(value, "$(VERSION)", version, -1)
		value = strings.Replace(value, "$(HOST)", fqdn.Get(), -1)

		m[kv[0]] = strings.ToLower(value)
	}
	return m, err
}

func (c *Config) setDefaults() {

	// Default AppLog.Timestamp to @timestamp if no value is entered
	// if c.AppLog.TimestampField == "" {
	// 	//if !c.MetaData.IsDefined("applog", "timestamp_field_name") {
	// 	c.AppLog.TimestampField = "@timestamp"
	// 	//}
	// }

	// // Default to @timestamp if no value entered
	// if c.Defaults.TimestampField == "" {
	// 	//if !c.MetaData.IsDefined("defaults", "timestamp_field_name") {
	// 	c.Defaults.TimestampField = "@timestamp"
	// 	//}
	// }

	// if c.Defaults.PayloadField == "" {
	// 	if !c.MetaData.IsDefined("defaults", "payload_field_name") {
	// 		c.Defaults.PayloadField = "mssql"
	// 	}
	// }

	// Start with the defaults
	// Then apply the settings from source if it has a value
	// Then replace the original source
	for i, v := range c.Sources {
		n := c.Defaults
		if v.FQDN != "" {
			n.FQDN = v.FQDN
		}
		if len(v.Sessions) > 0 {
			n.Sessions = v.Sessions
		}
		// if we are ignoring the sessions, set to empty array
		if v.IgnoreSessions {
			n.Sessions = []string{}
		}
		if len(v.ExcludedEvents) > 0 {
			n.ExcludedEvents = v.ExcludedEvents
		}

		if v.Prefix != "" {
			n.Prefix = v.Prefix
		}

		if v.AgentJobs != "" {
			n.AgentJobs = v.AgentJobs
		}

		if v.PayloadField != "" {
			n.PayloadField = v.PayloadField
		}

		if v.TimestampField != "" {
			n.TimestampField = v.TimestampField
		}

		if v.Rows != 0 {
			n.Rows = v.Rows
		}
		// if v.Test != false {
		// 	n.Test = v.Test
		// }
		// if v.Print != false {
		// 	n.Print = v.Print
		// }

		if len(v.Adds) > 0 {
			//n.Adds = v.Adds
			n.Adds = merge(n.Adds, v.Adds)
		}
		if len(v.Copies) > 0 {
			//n.Copies = v.Copies
			n.Copies = merge(n.Copies, v.Copies)
		}
		if len(v.Moves) > 0 {
			n.Moves = merge(n.Moves, v.Moves)
			// n.Renames = v.Renames
		}
		c.Sources[i] = n
	}
}

// merge takes base map (b) and adds the overrides to it
func merge(b, o map[string]string) map[string]string {
	m := make(map[string]string)
	for k, v := range b {
		m[k] = v
	}

	for k, v := range o {
		// if v is blank, remove the entry
		if len(v) == 0 {
			delete(m, k)
		} else {
			m[k] = v
		}

	}
	return m
}

// ToJSON returns a JSON string version of the configuration
func (s *Source) ToJSON() string {
	b, err := json.Marshal(s)
	if err != nil {
		return ""
	}
	return string(b)
}
