package sink

// Sinker defines places that events can be written
type Sinker interface {
	Open(string) error
	Write(string, string) (int, error)
	Flush() error
	Close() error
	Name() string
	Clean() error
	Reopen() error
}

// * filesink: file_name
// * logstash: host & port
