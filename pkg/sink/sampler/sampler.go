package sampler

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/billgraziano/xelogstash/pkg/sink"
	"github.com/spf13/afero"

	log "github.com/sirupsen/logrus"
)

// Sampler writes a sampling of events to JSON files for review
type Sampler struct {
	mu       sync.Mutex
	fs       afero.Fs
	clock    clock.Clock
	events   map[string]time.Time
	duration time.Duration
	path     string
	logger   *log.Entry
}

// New returns a new Sampler sink
func New(root string, duration time.Duration) *Sampler {
	if root == "" {
		root = "."
	}
	s := &Sampler{
		mu:       sync.Mutex{},
		fs:       afero.NewOsFs(),
		clock:    clock.New(),
		duration: duration,
		events:   make(map[string]time.Time),
		path:     filepath.Join(root, "sinks", "sampler"),
	}
	return s
}

// Open writes the directory structure for the Sampler sink
func (s *Sampler) Open(ctx context.Context, name string) error {
	return s.fs.MkdirAll(s.path, 0660)
}

// Write an event to a sample file every now and then
func (s *Sampler) Write(ctx context.Context, name, data string) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fileName := fmt.Sprintf("%s.json", name)
	lastWritten, exists := s.events[name]

	if exists {
		if s.clock.Now().Before(lastWritten.Add(s.duration)) {
			return len(data), nil
		}
		s.events[name] = s.clock.Now()
		return s.write(fileName, data)
	} else {
		s.events[name] = s.clock.Now()
		return s.write(fileName, data)
	}
}

func (s *Sampler) write(filename, data string) (int, error) {
	fullFile := filepath.Join(s.path, filename)
	if s.logger != nil {
		s.logger.Tracef("sampler: writing: %s", fullFile)
	}

	// reformat with pretty JSON
	var obj interface{}
	err := json.Unmarshal([]byte(data), &obj)
	if err != nil {
		return 0, nil
	}
	bb, err := json.MarshalIndent(obj, "", "\t")
	if err != nil {
		return 0, nil
	}

	err = afero.WriteFile(s.fs, fullFile, bb, 0660)
	return len(data), err
}

// Name returns a descriptive name for logging
func (s *Sampler) Name() string {
	return fmt.Sprintf("sampler: %s (%s)", s.path, s.duration.String())
}

func (s *Sampler) Close() error  { return nil }
func (s *Sampler) Sync() error   { return nil }
func (s *Sampler) Clean() error  { return nil }
func (s *Sampler) Reopen() error { return nil }
func (s *Sampler) Flush() error  { return nil }

func (s *Sampler) SetLogger(entry *log.Entry) { s.logger = entry }

// var _ io.WriteCloser = (*Sampler)(nil)
var _ sink.Sinker = (*Sampler)(nil)
