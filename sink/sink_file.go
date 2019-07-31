package sink

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// FileSink writes events to a file
type FileSink struct {
	file     *os.File
	id       string
	fileName string
	// sync.RWMutex
}

// NewFileSink returns a new FileSink
func NewFileSink() *FileSink {
	//ext := time.Now().Format("20060102_15")
	//fileName := fmt.Sprintf("xelogstash_events_%s.json", ext)
	// time.Parse("20060102150405", dt+tm)
	fs := FileSink{}
	return &fs
}

// Name returns the name of the sync
func (fs *FileSink) Name() string {
	// fs.RLock()
	// defer fs.RUnlock()
	//return fmt.Sprintf("filesink: %s", fs.name)
	return "filesink"
}

// Open opens the file
func (fs *FileSink) Open(id string) error {
	// fs.Lock()
	// defer fs.Unlock()
	return fs.open(id)
}

// Write a message to the FileSink
func (fs *FileSink) Write(msg string) (int, error) {
	var err error
	// fs.Lock()
	// defer fs.Unlock()

	if fs.file == nil {
		return 0, errors.New("filesink: not open")
	}
	if !strings.HasSuffix(msg, "\n") {
		msg = msg + "\n"
	}

	n, err := fs.file.Write([]byte(msg))
	if err != nil {
		return n, errors.Wrap(err, "fs.file.write")
	}
	return 0, nil
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
	fs.id = id
	fileName := makeFileName(id)
	executable, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "os.executable")
	}
	exeDir := filepath.Dir(executable)

	eventDir := filepath.Join(exeDir, "events")
	if _, err = os.Stat(eventDir); os.IsNotExist(err) {
		err = os.Mkdir(eventDir, 0644)
	}
	if err != nil {
		return errors.Wrap(err, "os.mkdir")
	}

	fqfile := filepath.Join(eventDir, fileName)
	lf, err := os.OpenFile(fqfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return errors.Wrap(err, "os.openfile")
	}
	fs.file = lf
	return nil
}

// Clean up any old artifacts
func (fs *FileSink) Clean() error {
	// fs.Lock()
	// defer fs.Unlock()

	var err error

	// tz := time.Now().Location()
	// hours := int(days*24 + 1)
	cutoff := 168 * time.Hour

	exe, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "os.executable")
	}
	exe = filepath.Base(exe)
	exe = strings.TrimSuffix(exe, path.Ext(exe))

	files, err := ioutil.ReadDir(filepath.Join(".", "events"))
	if err != nil {
		return errors.Wrap(err, "ioutil.readdir")
	}
	pattern := fmt.Sprintf("xelogstash_%s_d{8}_d{2}\\.json", fs.id)
	re := regexp.MustCompile(pattern)
	now := time.Now()

	for _, fi := range files {
		if fi.IsDir() {
			continue
		}

		m := re.FindStringSubmatch(fi.Name())
		if len(m) == 0 {
			continue
		}
		diff := now.Sub(fi.ModTime())
		if diff > cutoff {
			err = os.Remove(fi.Name())
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("os.remove: %s", fi.Name()))
			}
		}
	}
	return nil
}
