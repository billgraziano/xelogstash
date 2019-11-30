package rotator

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

// Rotator satisfies io.WriterCloser and is used to
// rotate log files and event files
type Rotator struct {
	Directory string
	Prefix    string
	Extension string
	Retention time.Duration
	Hourly    bool

	fs    afero.Fs
	clock clock.Clock
	file  afero.File
	mu    sync.Mutex
	ts    string
}

// New returns a new Rotator
func New(dir, prefix, extension string) *Rotator {
	r := &Rotator{
		Directory: dir,
		Prefix:    prefix,
		Extension: extension,
	}
	r.fs = afero.NewOsFs()
	r.clock = clock.New()
	r.Retention = time.Duration(168 * time.Hour)

	return r
}

// Start just runs the clean up process
func (r *Rotator) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.clean()
}

// Write a byte array to the log file
func (r *Rotator) Write(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.fs == nil {
		r.fs = afero.NewOsFs()
	}
	if r.clock == nil {
		r.clock = clock.New()
	}
	if r.Retention == 0 {
		r.Retention = time.Duration(170 * time.Hour)
	}

	// is this the first time we're writing?
	if r.ts == "" {
		r.ts = r.getts()
	}

	// do we have a file, if not, open one
	if r.file == nil {
		err = r.open()
		if err != nil {
			return 0, err
		}
	}

	// if the date timestamp changed, we are rotating
	ts := r.getts()
	if r.ts != ts {
		r.ts = ts
		err = r.rotate()
		if err != nil {
			return 0, err
		}
	}

	// write to the file
	n, err = r.file.Write(p)
	return n, err
}

func (r *Rotator) getts() string {
	if r.Hourly {
		return r.clock.Now().Format("20060102_15")
	}
	return r.clock.Now().Format("20060102")
}

func (r *Rotator) rotate() error {
	var err error
	err = r.close()
	if err != nil {
		return err
	}

	err = r.open()
	if err != nil {
		return err
	}

	err = r.clean()
	if err != nil {
		return err
	}
	return nil
}

func (r *Rotator) clean() error {
	// loop through all matching files
	// purge the old ones
	files, err := afero.ReadDir(r.fs, r.Directory)
	if err != nil {
		return fmt.Errorf("error reading directory: %s", err)
	}

	tz := time.Now().Location()

	re := regexp.MustCompile(fmt.Sprintf("%s_(?P<date>\\d{8})(_(?P<hour>\\d{2}))?\\.%s", r.Prefix, r.Extension))
	for _, fi := range files {
		if fi.IsDir() {
			continue
		}
		m := re.FindStringSubmatch(fi.Name())
		if len(m) == 0 {
			continue
		}
		if len(m) >= 1 {

			var fd time.Time
			fileDate := m[1]
			fileHour := m[1] + m[2]
			if m[2] == "" {
				fd, err = time.ParseInLocation("20060102", fileDate, tz)
				if err != nil {
					continue
				}
			} else {
				fd, err = time.ParseInLocation("20060102_15", fileHour, tz)
				if err != nil {
					continue
				}
			}

			if r.clock.Now().Sub(fd) > r.Retention {
				fqName := filepath.Join(r.Directory, fi.Name())
				err = r.fs.Remove(fqName)
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("s.fs.remove: %s", fi.Name()))
				}
			}
		}

	}
	return nil
}

func (r *Rotator) open() error {
	var err error
	if r.Directory != "" && r.Directory != "." {
		err := r.fs.MkdirAll(r.Directory, 0755)
		if err != nil {
			return fmt.Errorf("can't make directories for new logfile: %s", err)
		}
	}

	name := r.filename()
	mode := os.FileMode(0600)
	_, err = r.fs.Stat(name)
	if os.IsNotExist(err) {
		r.file, err = r.fs.OpenFile(name, os.O_CREATE|os.O_WRONLY, mode)
		if err != nil {
			return fmt.Errorf("can't create new logfile: %s", err)
		}
	} else {
		r.file, err = r.fs.OpenFile(name, os.O_APPEND|os.O_WRONLY, mode)
		if err != nil {
			return fmt.Errorf("can't open logfile: %s", err)
		}
	}

	return nil
}

func (r *Rotator) filename() string {
	name := fmt.Sprintf("%s_%s.%s", r.Prefix, r.ts, r.Extension)
	return filepath.Join(r.Directory, name)
}

// Close the current log file
func (r *Rotator) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.close()
}

func (r *Rotator) close() error {
	if r.file == nil {
		return nil
	}
	err := r.file.Close()
	if err != nil {
		return err
	}

	r.file = nil
	return err
}

func (r *Rotator) Sync() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.file != nil {
		return r.file.Sync()
	}
	return nil
}

var _ io.WriteCloser = (*Rotator)(nil)
