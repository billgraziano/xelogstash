package status

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

// key is used to define how a file is generated and preven duplicates
// type key struct {
// 	Prefix     string
// 	Type       string
// 	Instance   string
// 	Identifier string
// }

var sources map[string]bool
var mux sync.Mutex

//ErrDup indicates a duplicate was found
var ErrDup = errors.New("duplicate prefix-instance-class-id")

func init() {
	sources = make(map[string]bool)
	//mux = sync.Mutex
}

// File is a way to keep track of status
type File struct {
	// prefix         string
	// instance       string
	// session        string
	Name string
	file *os.File
}

const (
	// StatusSuccess means that we will read the file normally
	StatusSuccess = "good"

	// StatusReset means that we will assume a bad file name and offset
	StatusReset = "reset"
)

const (
	// ClassXE is used for XE sessions
	ClassXE = "XE"
	// ClassAgentJobs is used for AGENT job history
	ClassAgentJobs = "JOBS"
)

// CheckDupe checks to see if this session has been processed already
// If so, it returns an error
// It saves all sessions
// class is XE, AUDIT, or AGENT
// session is the XE session name or Audit session name
func CheckDupe(prefix, instance, class, id string) error {

	fileName := strings.ToLower(fileName(prefix, instance, class, id))

	mux.Lock()
	defer mux.Unlock()

	_, found := sources[fileName]
	if found {
		return ErrDup
	}

	sources[fileName] = true

	return nil
}

// NewFile generates a new status file for this prefix, instance, session
// This also creates the status file if it doesn't exist
func NewFile(prefix, instance, class, id string) (File, error) {
	var f File
	var err error

	// Get EXE directory
	executable, err := os.Executable()
	if err != nil {
		return f, errors.Wrap(err, "os.executable")
	}
	exeDir := filepath.Dir(executable)

	statusDir := filepath.Join(exeDir, "status")
	if _, err = os.Stat(statusDir); os.IsNotExist(err) {
		err = os.Mkdir(statusDir, 0644)
	}
	if err != nil {
		return f, errors.Wrap(err, "os.mkdir")
	}

	f.Name = filepath.Join(statusDir, fileName(prefix, instance, class, id))
	return f, nil
}

// GetOffset returns the last file and offset for this file status
func (f *File) GetOffset() (fileName string, offset int64, xestatus string, err error) {

	_, err = os.Stat(f.Name)
	if os.IsNotExist(err) {
		fp, err := os.OpenFile(f.Name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return "", 0, StatusReset, errors.Wrap(err, "create")
		}
		f.file = fp
		return "", 0, StatusSuccess, nil
	} else if err != nil {
		return "", 0, StatusReset, errors.Wrap(err, "stat")
	}

	readonly, err := os.OpenFile(f.Name, os.O_RDONLY, 0666)
	if err != nil {
		return "", 0, StatusReset, errors.Wrap(err, "openreadonly")
	}

	reader := csv.NewReader(bufio.NewReader(readonly))
	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return "", 0, StatusReset, errors.Wrap(err, "read")
		}
		if len(line) < 2 || len(line) > 3 {
			return "", 0, StatusReset, errors.Errorf("len(line) expected: 2 or 3; got %d (%v)", len(line), line)
		}

		fileName = strings.TrimSpace(line[0])
		offset, err = strconv.ParseInt(strings.TrimSpace(line[1]), 10, 64)
		if err != nil {
			return "", 0, StatusReset, errors.Errorf("error reading offset: got %s", line[1])
		}
		if len(line) == 2 {
			xestatus = StatusSuccess // Assume we are good
		} else {
			xestatus = strings.TrimSpace(line[2])
		}
	}
	err = readonly.Close()
	if err != nil {
		return "", 0, StatusReset, errors.Wrap(err, "close")
	}

	// TODO close & reopen the file
	//log.Println("I'm opening for append")
	fp, err := os.OpenFile(f.Name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return "", 0, StatusReset, errors.Wrap(err, "openappend")
	}
	//log.Println("setup name: ", f.Name)

	_, err = fp.Stat()
	if err != nil {
		return fileName, offset, StatusReset, errors.Wrap(err, "stat-2")
	}

	f.file = fp

	return fileName, offset, xestatus, nil
}

// FileName returns the base file name to track status
func fileName(prefix, instance, class, id string) string {
	var fileName string
	instance = strings.Replace(instance, "\\", "__", -1)

	// use it to build the file name
	if prefix == "" {
		fileName = fmt.Sprintf("%s_%s_%s.status", instance, class, id)
	} else {
		fileName = fmt.Sprintf("%s_%s_%s_%s.status", prefix, instance, class, id)
	}
	return fileName
}

// Save persists the last filename and offset that was successfully completed
func (f *File) Save(fileName string, offset int64, xestatus string) error {
	if f.file == nil {
		return errors.New("status file not open")
	}

	err := writeStatus(f.file, fileName, offset, xestatus)
	if err != nil {
		return errors.Wrap(err, "writeStatus")
	}

	return nil
}

func writeStatus(f *os.File, xeFileName string, offset int64, xestatus string) error {
	msg := fmt.Sprintf("%s, %d, %s\r\n", xeFileName, offset, xestatus)
	_, err := f.WriteString(msg)
	if err != nil {
		return errors.Wrap(err, "file.write")
	}
	return nil
}

// Done closes the file
func (f *File) Done(xeFileName string, offset int64, xestatus string) error {
	var err error
	err = f.Save(xeFileName, offset, xestatus)
	if err != nil {
		return errors.Wrap(err, "save")
	}

	err = f.file.Close()
	if err != nil {
		return errors.Wrap(err, "close")
	}

	if f.Name == "" {
		return errors.New("f.Name is empty")
	}

	// Delete the .0 file
	safetyFileName := fmt.Sprintf("%s.0", f.Name)
	_, err = os.Stat(safetyFileName)
	if err != nil {
		if !os.IsNotExist(err) {
			return errors.Wrap(err, "stat")
		}
	}

	if os.IsExist(err) {
		err = os.Remove(safetyFileName)
		if err != nil {
			return errors.Wrap(err, "remove")
		}
	}

	// Rename to the .0 file
	err = os.Rename(f.Name, safetyFileName)
	if err != nil {
		return errors.Wrap(err, "rename")
	}

	// Write the new file
	newStatusFile, err := os.OpenFile(f.Name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return errors.Wrap(err, "create")
	}

	err = writeStatus(newStatusFile, xeFileName, offset, xestatus)
	if err != nil {
		return errors.Wrap(err, "writestatus")
	}

	err = newStatusFile.Close()
	if err != nil {
		return errors.Wrap(err, "close")
	}

	return nil
}
