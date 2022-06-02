package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/billgraziano/xelogstash/sink"

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

// DefaultStopAt is the date we use for stop at if not defined
var DefaultStopAt = time.Date(9999, time.December, 31, 0, 0, 0, 0, time.UTC)

func sourcesFile(f string) (cfg Config, err error) {
	_, err = toml.DecodeFile(f, &cfg)
	if err != nil {
		return cfg, errors.Wrap(err, "toml.decode")
	}
	return cfg, nil
}

// Get the configuration from a configuration file
func Get(f, src, version, sha1ver string) (config Config, err error) {
	config.FileName = f
	config.SourcesFile = src
	md, err := toml.DecodeFile(f, &config)
	if err != nil {
		return config, errors.Wrap(err, "toml.decode")
	}

	config.MetaData = md

	// Get the extra sources
	// add the sources before config.decodekv fixes up the adds, etc.
	if src != "" {
		sources, err := sourcesFile(src)
		if err != nil {
			return config, errors.Wrap(err, "sourcesfile")
		}
		config.Sources = append(config.Sources, sources.Sources...)
	}

	err = config.decodekv(version, sha1ver)
	if err != nil {
		return config, errors.Wrap(err, "decodekv")
	}

	if config.Defaults.StopAt.IsZero() {
		config.Defaults.StopAt = DefaultStopAt
	}

	if config.Defaults.PollSeconds == 0 {
		config.Defaults.PollSeconds = 60
	}

	if config.App.HTTPMetricsPort == 0 {
		config.App.HTTPMetricsPort = 8080
	}
	// Calculate the default lookback and use if more recent than StartAt
	if config.Defaults.LookBackRaw != "" {
		err = config.Defaults.processLookback()
		if err != nil {
			return config, errors.Wrap(err, "source.proceslookback")
		}
	}

	// make sure each filter has a filter action field
	for i, f := range config.Filters {
		_, ok := f["filter_action"]
		if !ok {
			return config, fmt.Errorf("filter #%d missing 'filter_action'", i+1)
		}
	}

	err = config.Defaults.validate()
	if err != nil {
		return config, errors.Wrap(err, "config.defaults.validate")
	}

	err = config.setSourceDefaults()
	if err != nil {
		return config, errors.Wrap(err, "config.setsourcedefaults")
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

	// if config.Sinks == nil {
	// 	config.Sinks = make([]sink.Sinker, 0)
	// }

	// Set FileSink defaults
	if config.FileSink != nil {
		if config.FileSink.RetainHours == 0 {
			config.FileSink.RetainHours = 168 // 7 days
		}

		if config.FileSink.Directory == "" {
			var executable string
			executable, err = os.Executable()
			if err != nil {
				return config, errors.Wrap(err, "os.executable")
			}
			exeDir := filepath.Dir(executable)
			config.FileSink.Directory = filepath.Join(exeDir, "events")
		}

		rot := sink.NewRotator(config.FileSink.Directory, "sqlevents", "events")
		rot.Retention = time.Duration(config.FileSink.RetainHours) * time.Hour
		rot.Hourly = true
		config.rot = rot
	}

	return config, err
}

// CloseRotator closes the rotator stored in a Config
func (c *Config) CloseRotator() error {
	if c.rot == nil {
		return nil
	}
	return c.rot.Close()
}

func (c *Config) GetRotator() *sink.Rotator {
	return c.rot
}

// GetSinks returns an array of sinks based on the config
func (c *Config) GetSinks() ([]sink.Sinker, error) {
	sinks := make([]sink.Sinker, 0)

	// Add FileSink
	if c.FileSink != nil {
		//fileSink := sink.NewFileSink(c.FileSink.Directory, c.FileSink.RetainHours)
		of := sink.NewOneFile(c.rot)
		sinks = append(sinks, of)
	}

	// Add an ElasticSink
	if len(c.Elastic.Addresses) > 0 && c.Elastic.Username != "" && c.Elastic.Password != "" {
		es, err := sink.NewElasticSink(c.Elastic.Addresses, c.Elastic.ProxyServer, c.Elastic.Username, c.Elastic.Password)
		if err != nil {
			return sinks, errors.Wrap(err, "sink.newelasticsink")
		}
		es.DefaultIndex = c.Elastic.DefaultIndex
		es.EventIndexMap, err = buildmap(c.Elastic.RawEventMap, "", "")
		if err != nil {
			return sinks, errors.Wrap(err, "elastic.buildmap")
		}
		es.AutoCreateIndexes = c.Elastic.AutoCreateIndexes
		sinks = append(sinks, es)
	}

	// Add LogstashSink
	if c.Logstash != nil {
		ls := *c.Logstash
		lss, err := sink.NewLogstashSink(ls.Host, 30)
		if err != nil {
			return sinks, errors.Wrap(err, "sink.newlogstashsink")
		}
		//lss.RetryAlertThreshold = c.Logstash.RetryAlertThreshold
		sinks = append(sinks, lss)
	}

	return sinks, nil
}

// processLookBack pushes the StartAt forward if needed based on look_back
func (s *Source) processLookback() error {
	if s.LookBackRaw == "" {
		return nil
	}
	lb, err := time.ParseDuration(s.LookBackRaw)
	if err != nil {
		return errors.Wrapf(err, "invalid lookback: %s", s.LookBackRaw)
	}
	lookBackAbs := time.Now().Add(-1 * lb)
	if lookBackAbs.After(s.StartAt) {
		s.StartAt = lookBackAbs
	}
	return nil
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

// func (c *Config) setLowerCase() {
// 	// excluded events
// 	for i := range c.Defaults.ExcludedEvents {
// 		c.Defaults.ExcludedEvents[i] = strings.ToLower(c.Defaults.ExcludedEvents[i])
// 	}

// 	for j := range c.Sources {
// 		for i := range c.Sources[j].ExcludedEvents {
// 			c.Sources[j].ExcludedEvents[i] = strings.ToLower(c.Sources[j].ExcludedEvents[i])
// 		}
// 	}
// }

func (c *Config) decodekv(version, sha1ver string) error {
	var err error
	c.Defaults.Adds, err = buildmap(c.Defaults.RawAdds, version, sha1ver)
	if err != nil {
		return errors.Wrap(err, "default-adds")
	}
	if c.Defaults.Copies, err = buildmap(c.Defaults.RawCopies, version, sha1ver); err != nil {
		return errors.Wrap(err, "default-copies")
	}
	if c.Defaults.Moves, err = buildmap(c.Defaults.RawMoves, version, sha1ver); err != nil {
		return errors.Wrap(err, "default-renames")
	}

	for i := range c.Sources {
		if c.Sources[i].Adds, err = buildmap(c.Sources[i].RawAdds, version, sha1ver); err != nil {
			return errors.Wrap(err, "source-adds")
		}
		if c.Sources[i].Copies, err = buildmap(c.Sources[i].RawCopies, version, sha1ver); err != nil {
			return errors.Wrap(err, "source-copies")
		}
		if c.Sources[i].Moves, err = buildmap(c.Sources[i].RawMoves, version, sha1ver); err != nil {
			return errors.Wrap(err, "source-renames")
		}
	}

	c.Elastic.EventIndexMap, err = buildmap(c.Elastic.RawEventMap, version, sha1ver)
	if err != nil {
		return errors.Wrap(err, "buildmap.elastic.eventindexmap")
	}

	// if c.App.Copies, err = buildmap(c.App.RawCopies); err != nil {
	// 	return errors.Wrap(err, "app-copies")
	// }

	// if c.App.Renames, err = buildmap(c.App.RawMoves); err != nil {
	// 	return errors.Wrap(err, "app-renames")
	// }

	return err
}

// buildmap converts key:value to a map[string]string and replaces variables
func buildmap(a []string, version, sha1ver string) (map[string]string, error) {
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

		value = strings.Replace(value, "$(EXENAMEPATH)", strings.ToLower(exeNamePath), -1)
		value = strings.Replace(value, "$(EXENAME)", strings.ToLower(exeName), -1)
		value = strings.Replace(value, "$(PID)", strconv.Itoa(os.Getpid()), -1)
		value = strings.Replace(value, "$(VERSION)", version, -1)
		value = strings.Replace(value, "$(GITDESCRIBE)", version, -1)
		value = strings.Replace(value, "$(GITHASH)", sha1ver, -1)
		value = strings.Replace(value, "$(HOST)", strings.ToLower(fqdn.Get()), -1)

		m[kv[0]] = value
	}
	return m, err
}

func (c *Config) setSourceDefaults() error {

	// Start with the defaults
	// Then apply the settings from source if it has a value
	// Then replace the original source
	for i, v := range c.Sources {
		var err error
		n := c.Defaults

		if v.FQDN != "" {
			n.FQDN = v.FQDN
		}
		if v.User != "" {
			n.User = v.User
		}
		if v.Password != "" {
			n.Password = v.Password
		}
		if v.ServerNameOverride != "" {
			n.ServerNameOverride = v.ServerNameOverride
		}
		if v.DomainNameOverride != "" {
			n.DomainNameOverride = v.DomainNameOverride
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

		if v.PollSeconds > 0 {
			n.PollSeconds = v.PollSeconds
		}

		if !v.StartAt.IsZero() {
			n.StartAt = v.StartAt
		}

		if v.LookBackRaw != "" {
			n.LookBackRaw = v.LookBackRaw
		}

		err = n.processLookback()
		if err != nil {
			return errors.Wrapf(err, "invalid look_back: %s", v.LookBackRaw)
		}

		if !v.StopAt.IsZero() {
			n.StopAt = v.StopAt
		}

		if v.Exclude17830 {
			n.Exclude17830 = v.Exclude17830
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
	return nil
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

// Print the values of elastic config
func (e *ElasticConfig) Print() {
	fmt.Println("-- elastic config --------------------------------")
	for i, v := range e.Addresses {
		fmt.Printf("-- address[%d]: %s\r\n", i, v)
	}
	fmt.Println("-- username:", e.Username)
	fmt.Println("-- default_index:", e.DefaultIndex)
	for k, v := range e.EventIndexMap {
		fmt.Printf("-- event: %s -> %s\r\n", k, v)
	}
	fmt.Println("--------------------------------------------------")

}
