package status

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	// "github.com/billgraziano/xelogstash/log"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// key is used to define how a file is generated and preven duplicates
// type key struct {
// 	Prefix     string
// 	Type       string
// 	Instance   string
// 	Identifier string
// }

var sources map[string]bool
var instances map[string]bool
var mux sync.Mutex

//ErrDup indicates a duplicate was found
var ErrDup = errors.New("duplicate domain-instance-class-id")

// ErrDupInstance indicates a duplicate instance domain combination was found
var ErrDupInstance = errors.New("duplicate domain-instance")

func init() {
	sources = make(map[string]bool)
	instances = make(map[string]bool)
	//mux = sync.Mutex
}

// File is a way to keep track of state
type File struct {
	// prefix         string
	// instance       string
	// session        string
	Name string
	file *os.File
}

const (
	// StateSuccess means that we will read the file normally
	StateSuccess = "good"

	// StateReset means that we will assume a bad file name and offset
	StateReset = "reset"
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
func CheckDupe(domain, instance, class, id string) error {

	fileName := strings.ToLower(fileName(domain, instance, class, id))

	mux.Lock()
	defer mux.Unlock()

	_, found := sources[fileName]
	if found {
		return ErrDup
	}

	sources[fileName] = true

	return nil
}

// CheckDupeInstance checks to see if this instance has already been seen
func CheckDupeInstance(domain, instance string) error {
	if domain == "" || instance == "" {
		return fmt.Errorf("invalid value: domain: %s; instance: %s;", domain, instance)
	}
	key := domain + ":" + instance
	mux.Lock()
	defer mux.Unlock()

	_, found := instances[key]
	if found {
		return ErrDupInstance
	}
	instances[key] = true
	return nil
}

// Reset clears out the list of servers so we can restart
func Reset() {
	sources = make(map[string]bool)
	instances = make(map[string]bool)
}

// NewFile generates a new state file for this domain, instance, session
// This also creates the state file if it doesn't exist
func NewFile(domain, instance, class, id string) (File, error) {
	var f File
	var err error

	// Get EXE directory
	executable, err := os.Executable()
	if err != nil {
		return f, errors.Wrap(err, "os.executable")
	}
	exeDir := filepath.Dir(executable)

	stateDir := filepath.Join(exeDir, "xestate")
	if _, err = os.Stat(stateDir); os.IsNotExist(err) {
		err = os.Mkdir(stateDir, 0644)
	}
	if err != nil {
		return f, errors.Wrap(err, "os.mkdir")
	}

	f.Name = filepath.Join(stateDir, fileName(domain, instance, class, id))
	return f, nil
}

// checkNullFile checks if the state file contains only 0x0's
func (f *File) checkNullFile() error {
	var err error
	_, err = os.Stat(f.Name)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return errors.Wrap(err, "os.stat")
	}
	bb, err := ioutil.ReadFile(f.Name)
	if err != nil {
		return errors.Wrap(err, "ioutil.readfile")
	}
	if len(bb) == 0 {
		return nil
	}

	// if any byte isn't zero, exit
	for _, b := range bb {
		if b != 0 {
			return nil
		}
	}

	// file is all nulls
	return errors.New(fmt.Sprintf("state file all nulls (replace with .0 file): %s", f.Name))

	// delete the file - commented out because we don't automatically check the .0 file
	// log.Warnf("state file all nulls: removing: %s", f.Name)
	// err = os.Remove(f.Name)
	// if err != nil {
	// 	return errors.Wrap(err, "os.remove")
	// }
}

// GetOffset returns the last file and offset for this file state
func (f *File) GetOffset() (fileName string, offset int64, xestatus string, err error) {

	err = f.checkNullFile()
	if err != nil {
		return "", 0, StateReset, errors.Wrap(err, "checknulls")
	}
	var fp *os.File
	_, err = os.Stat(f.Name)
	if os.IsNotExist(err) {
		fp, err = os.OpenFile(f.Name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return "", 0, StateReset, errors.Wrap(err, "create")
		}
		f.file = fp
		return "", 0, StateSuccess, nil
	} else if err != nil {
		return "", 0, StateReset, errors.Wrap(err, "stat")
	}

	readonly, err := os.OpenFile(f.Name, os.O_RDONLY, 0666)
	if err != nil {
		return "", 0, StateReset, errors.Wrap(err, "openreadonly")
	}

	var line []string
	reader := csv.NewReader(bufio.NewReader(readonly))
	for {
		line, err = reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return "", 0, StateReset, errors.Wrap(err, "read")
		}
		if len(line) < 2 || len(line) > 3 {
			return "", 0, StateReset, errors.Errorf("len(line) expected: 2 or 3; got %d (%v)", len(line), line)
		}

		fileName = strings.TrimSpace(line[0])
		offset, err = strconv.ParseInt(strings.TrimSpace(line[1]), 10, 64)
		if err != nil {
			return "", 0, StateReset, errors.Errorf("error reading offset: got %s", line[1])
		}
		if len(line) == 2 {
			xestatus = StateSuccess // Assume we are good
		} else {
			xestatus = strings.TrimSpace(line[2])
		}
	}
	err = readonly.Close()
	if err != nil {
		return "", 0, StateReset, errors.Wrap(err, "close")
	}

	// TODO close & reopen the file
	fp, err = os.OpenFile(f.Name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return "", 0, StateReset, errors.Wrap(err, "openappend")
	}

	_, err = fp.Stat()
	if err != nil {
		return fileName, offset, StateReset, errors.Wrap(err, "stat-2")
	}

	f.file = fp

	return fileName, offset, xestatus, nil
}

// FileName returns the base file name to track state
func fileName(domain, instance, class, id string) string {
	var fileName string
	instance = strings.Replace(instance, "\\", "__", -1)

	// use it to build the file name
	// if prefix == "" {
	// 	fileName = fmt.Sprintf("%s_%s_%s.state", instance, class, id)
	// } else {
	fileName = fmt.Sprintf("%s_%s_%s_%s.state", domain, instance, class, id)
	//}
	return fileName
}

// FileName returns the base file name to track status
func legacyFileName(prefix, instance, class, id string) string {
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
		return errors.New("state file not open")
	}

	err := writeState(f.file, fileName, offset, xestatus)
	if err != nil {
		return errors.Wrap(err, "writeStatus")
	}

	return nil
}

func writeState(f *os.File, xeFileName string, offset int64, xestatus string) error {
	msg := fmt.Sprintf("%s, %d, %s\r\n", xeFileName, offset, xestatus)
	_, err := f.WriteString(msg)
	if err != nil {
		return errors.Wrap(err, "file.write")
	}
	err = f.Sync()
	if err != nil {
		return errors.Wrap(err, "f.sync")
	}
	return nil
}

// Done closes the file
func (f *File) Done(xeFileName string, offset int64, xestatus string) error {
	var err error
	err = f.Save(xeFileName, offset, xestatus)
	if err != nil {
		return errors.Wrap(err, "f.save")
	}
	err = f.file.Sync()
	if err != nil {
		return errors.Wrap(err, "f.file.sync")
	}
	err = f.file.Close()
	if err != nil {
		return errors.Wrap(err, "f.file.close")
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

	err = writeState(newStatusFile, xeFileName, offset, xestatus)
	if err != nil {
		return errors.Wrap(err, "writestate")
	}

	err = newStatusFile.Sync()
	if err != nil {
		return errors.Wrap(err, "newstatusfile.sync")
	}

	err = newStatusFile.Close()
	if err != nil {
		return errors.Wrap(err, "close")
	}

	return nil
}

// SwitchV2 moves to new dir and name scheme
func SwitchV2(wid int, prefix, domain, instance, class, session string) error {
	mux.Lock()
	defer mux.Unlock()
	var msg string
	/*
		1. Get old file name
		2. Get new file name
		3. move the file
	*/

	// Get EXE directory
	executable, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "os.executable")
	}
	exeDir := filepath.Dir(executable)

	legacyDir := filepath.Join(exeDir, "status")
	legacyFile := filepath.Join(legacyDir, legacyFileName(prefix, instance, class, session))

	// if old dir (/status) doesn't exist, we're done
	_, err = os.Stat(legacyDir)
	if os.IsNotExist(err) {
		return nil
	}

	// does the old file exist?
	_, err = os.Stat(legacyFile)
	if os.IsNotExist(err) {
		return nil
	}

	log.Debug(fmt.Sprintf("[%d] Legacy status file: %s", wid, legacyFile))
	newDir := filepath.Join(exeDir, "xestate")
	newFile := filepath.Join(newDir, fileName(domain, instance, class, session))

	// make the new state directory if it doesn't exist
	if _, err = os.Stat(newDir); os.IsNotExist(err) {
		msg = fmt.Sprintf("[%d] Making new state directory: %s", wid, newDir)
		log.Info(msg)
		err = os.Mkdir(newDir, 0666)
	}
	if err != nil {
		return errors.Wrap(err, "os.mkdir")
	}

	// does the new file exist?
	_, err = os.Stat(newFile)
	if !os.IsNotExist(err) {
		return fmt.Errorf("NEW STATE FILE ALREADY EXISTS: %s", newFile)
	}

	// Move the file
	msg = fmt.Sprintf("[%d] Moving %s\\%s to %s\\%s", wid, "status", filepath.Base(legacyFile), "xestate", filepath.Base(newFile))
	log.Info(msg)
	err = os.Rename(legacyFile, newFile)
	if err != nil {
		return errors.Wrap(err, "os.rename")
	}

	// Remove the .0 file
	zeroFile := legacyFile + ".0"
	_, err = os.Stat(zeroFile)
	if !os.IsNotExist(err) {
		msg = fmt.Sprintf("[%d] Removing temp file %s\\%s", wid, "status", filepath.Base(zeroFile))
		log.Info(msg)
		err = os.Remove(zeroFile)
		if err != nil {
			return errors.Wrap(err, "os.remove")
		}
	}
	return nil
}
