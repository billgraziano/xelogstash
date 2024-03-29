package config

import (
	"time"

	"github.com/billgraziano/toml"
	"github.com/billgraziano/xelogstash/pkg/sink"
)

// Config defines the configuration read from the TOML file
type Config struct {
	App      App
	AppLog   *AppLog
	Defaults Source   `toml:"defaults"`
	Sources  []Source `toml:"source"`

	Elastic  ElasticConfig `toml:"elastic"`
	FileSink *FileSink     `toml:"filesink"`
	Logstash *Logstash     `toml:"logstash"`
	Sampler  *Sampler      `toml:"sampler"`
	MetaData toml.MetaData

	ConfigFile     string
	ConfigFileMod  time.Time
	SourcesFile    string
	SourcesFileMod time.Time

	Filters []Filter `toml:"filter"`
	rot     *sink.Rotator
	//Sinks    []sink.Sinker
}

// Source defines a source of extended event information
type Source struct {
	FQDN               string
	User               string `toml:"user"`
	Password           string `toml:"password"`
	Driver             string `toml:"driver"`
	ODBCDriver         string `toml:"odbc_driver"`
	ServerNameOverride string `toml:"server_name_override"`
	DomainNameOverride string `toml:"domain_name_override"`
	PollSeconds        int    `toml:"poll_seconds"`

	Sessions       []string
	IgnoreSessions bool `toml:"ignore_sessions"` // if true, skip XE sessions
	Prefix         string
	AgentJobs      string
	PayloadField   string `toml:"payload_field_name"`
	TimestampField string `toml:"timestamp_field_name"`
	Rows           int
	StripCRLF      bool      `toml:"strip_crlf"`
	StartAt        time.Time `toml:"start_at"`
	StopAt         time.Time `toml:"stop_at"`
	LookBackRaw    string    `toml:"look_back"`

	Adds               map[string]string
	Copies             map[string]string
	Moves              map[string]string
	ExcludedEvents     []string //XE events that are excluded.  Mostly from system health
	Exclude17830       bool     `toml:"exclude_17830"`
	LogBadXML          bool     `toml:"log_bad_xml"`
	IncludeDebugDLLMsg bool     `toml:"include_dbghelpdll_msg"`

	RawAdds         []string `toml:"adds"`
	RawCopies       []string `toml:"copies"`
	RawMoves        []string `toml:"moves"`
	UppercaseFields []string `toml:"uppercase"`
	LowercaseFields []string `toml:"lowercase"`
}

// App defines the application configuration
type App struct {
	Workers        int
	Logstash       string
	Samples        bool   // Print sample JSON to stdout
	Summary        bool   // Print a summary to stdout
	LogLevel       string `toml:"log_level"` // trace|debug|info|...
	Verbose        bool   // Enables logging wrote x events at the info level
	StrictSessions bool   `toml:"strict_sessions"` // true - log session errors
	WatchConfig    bool   `toml:"watch_config"`
	BetaFeatures   bool   `toml:"beta_features"`

	// Enables a web server on :8080 with basic metrics
	HTTPMetrics     bool `toml:"http_metrics"`
	HTTPMetricsPort int  `toml:"http_metrics_port"`
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

// ElasticConfig holds the configuration for sending events to elastic
type ElasticConfig struct {
	Addresses         []string `toml:"addresses"`
	Username          string   `toml:"username"`
	Password          string   `toml:"password"`
	DefaultIndex      string   `toml:"default_index"`
	AppLogIndex       string   `toml:"applog_index"`
	EventIndexMap     map[string]string
	RawEventMap       []string `toml:"event_index_map"`
	AutoCreateIndexes bool     `toml:"auto_create_indexes"`
	ProxyServer       string   `toml:"proxy_server"`
}

// FileSink configures a file sink
type FileSink struct {
	Directory   string `toml:"dir"`
	RetainHours int    `toml:"retain_hours"`
}

// Logstash configures a LogstashSink
type Logstash struct {
	Host                string `toml:"host"`
	RetryAlertThreshold int    `toml:"retry_alert_threshold"`
}

type duration struct {
	time.Duration
}

func (d *duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

// Sampler configures a SamplerSink
type Sampler struct {
	Duration duration
}

type Filter map[string]interface{}
