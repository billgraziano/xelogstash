package sink

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// OneFile writes events to a file.  This is just a wrapper for the Rotator.
type OneFile struct {
	r *Rotator
}

// NewOneFile returns a OneFile
func NewOneFile(rot *Rotator) *OneFile {
	return &OneFile{r: rot}
}

// Name returns the name of the sink.  This is used for logging.
func (fs *OneFile) Name() string {
	var n string
	if fs.r.Hourly {
		n = fmt.Sprintf("%s_YYYYMMDD_HH.%s", fs.r.Prefix, fs.r.Extension)
	} else {
		n = fmt.Sprintf("%s_YYYYMMDD_.%s", fs.r.Prefix, fs.r.Extension)
	}
	name := fmt.Sprintf("files: %s (keep: %s)", filepath.Join(fs.r.Directory, n), fs.r.Retention.String())
	return name
	// return fmt.Sprintf("files: %s\\%s_YYYYMMDD_.%s (keep: %s)", fs.r.Directory, fs.r.Prefix, fs.r.Extension, fs.r.Retention.String())
}

// Open opens the file.  This is done in the Rotator.
func (fs *OneFile) Open(_ context.Context, _ string) error {
	return nil
}

// Write a message to the OneFile
func (fs *OneFile) Write(ctx context.Context, name, event string) (int, error) {
	var err error

	if !strings.HasSuffix(event, "\n") {
		event = event + "\n"
	}

	n, err := fs.r.Write([]byte(event))
	if err != nil {
		return n, errors.Wrap(err, "fs.file.write")
	}
	return n, nil
}

// Close the OneFile
func (fs *OneFile) Close() error {
	return fs.r.Sync()
}

// Flush any buffers
func (fs *OneFile) Flush() error {
	return fs.r.Sync()
}

// Clean up any old artifacts.  This is done in the Rotator.
func (fs *OneFile) Clean() error {
	return nil
}

// Reopen is a noop at this point
func (fs *OneFile) Reopen() error {
	return nil
}

// SetLogger sets the logger for the sink
func (fs *OneFile) SetLogger(entry *log.Entry) {
}
