package sink

import (
	"fmt"
	"strings"

	"github.com/billgraziano/xelogstash/pkg/rotator"
	"github.com/pkg/errors"
)

// OneFile writes events to a file.  This is just a wrapper for the Rotator.
type OneFile struct {
	r *rotator.Rotator
}

// NewOneFile returns a OneFile
func NewOneFile(rot *rotator.Rotator) *OneFile {
	return &OneFile{r: rot}
}

// Name returns the name of the sink.  This is used for logging.
func (fs *OneFile) Name() string {
	if fs.r.Hourly {
		return fmt.Sprintf("files: %s\\%s_YYYYMMDD_HH.%s (keep: %s)", fs.r.Directory, fs.r.Prefix, fs.r.Extension, fs.r.Retention.String())
	}
	return fmt.Sprintf("files: %s\\%s_YYYYMMDD_.%s (keep: %s)", fs.r.Directory, fs.r.Prefix, fs.r.Extension, fs.r.Retention.String())
}

// Open opens the file.  This is done in the Rotator.
func (fs *OneFile) Open(id string) error {
	return nil
}

// Write a message to the OneFile
func (fs *OneFile) Write(name, event string) (int, error) {
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
