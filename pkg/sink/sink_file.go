package sink

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// FileSink writes events to a file
type FileSink struct {
	file           *os.File
	id             string
	Directory      string
	RetentionHours int
	// sync.RWMutex
}

// NewFileSink returns a new FileSink
func NewFileSink(dir string, retain int) *FileSink {
	//ext := time.Now().Format("20060102_15")
	//fileName := fmt.Sprintf("xelogstash_events_%s.json", ext)
	// time.Parse("20060102150405", dt+tm)
	fs := FileSink{Directory: dir, RetentionHours: retain}
	return &fs
}

// Name returns the name of the sink
func (fs *FileSink) Name() string {
	// fs.RLock()
	// defer fs.RUnlock()
	//return fmt.Sprintf("filesink: %s", fs.name)
	return fmt.Sprintf("filesink: %s (retain: %d hours)", fs.Directory, fs.RetentionHours)
}

// Open opens the file
func (fs *FileSink) Open(id string) error {
	// fs.Lock()
	// defer fs.Unlock()
	return fs.open(id)
}

// Write a message to the FileSink
func (fs *FileSink) Write(name, event string) (int, error) {
	var err error
	// fs.Lock()
	// defer fs.Unlock()

	if fs.file == nil {
		return 0, errors.New("filesink: not open")
	}
	if !strings.HasSuffix(event, "\n") {
		event = event + "\n"
	}

	n, err := fs.file.Write([]byte(event))
	if err != nil {
		return n, errors.Wrap(err, "fs.file.write")
	}
	return n, nil
}

// Close the FileSink
func (fs *FileSink) Close() error {
	// fs.Lock()
	// defer fs.Unlock()

	if fs.file == nil {
		return nil
	}

	// TODO handle this error
	_ = fs.file.Close()

	// TODO clean up old files
	return nil
}

// Flush any buffers
func (fs *FileSink) Flush() error {
	// fs.Lock()
	// defer fs.Unlock()

	if fs.file == nil {
		return nil
	}
	return fs.file.Sync()
}
func makeFileName(id string) string {
	ext := time.Now().Format("20060102_15")
	fileName := fmt.Sprintf("xelogstash_%s_%s.json", id, ext)
	return fileName
}

func (fs *FileSink) open(id string) error {
	var err error
	fs.id = id
	fileName := makeFileName(id)
	// executable, err := os.Executable()
	// if err != nil {
	// 	return errors.Wrap(err, "os.executable")
	// }
	// exeDir := filepath.Dir(executable)

	// eventDir := filepath.Join(exeDir, "events")
	eventDir := fs.Directory
	if _, err = os.Stat(eventDir); os.IsNotExist(err) {
		err = os.Mkdir(eventDir, 0644)
	}
	if err != nil {
		return errors.Wrap(err, "os.mkdir")
	}

	fqfile := filepath.Join(eventDir, fileName)
	lf, err := os.OpenFile(filepath.Clean(fqfile), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return errors.Wrap(err, "os.openfile")
	}
	fs.file = lf
	return nil
}

// Clean up any old artifacts
func (fs *FileSink) Clean() error {
	var err error

	cutoff := time.Duration(fs.RetentionHours) * time.Hour
	files, err := os.ReadDir(fs.Directory)
	if err != nil {
		return errors.Wrap(err, "os.readdir")
	}
	pattern := fmt.Sprintf("xelogstash_%s_\\d{8}_\\d{2}\\.json", fs.id)
	re := regexp.MustCompile(pattern)
	now := time.Now()

	for _, di := range files {
		fi, err := di.Info()
		if err != nil {
			return err
		}
		if fi.IsDir() {
			continue
		}
		m := re.FindStringSubmatch(fi.Name())
		if len(m) == 0 {
			continue
		}
		diff := now.Sub(fi.ModTime())
		if diff > cutoff {
			err = os.Remove(filepath.Join(fs.Directory, fi.Name()))
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("os.remove: %s", fi.Name()))
			}
		}
	}
	return nil
}
