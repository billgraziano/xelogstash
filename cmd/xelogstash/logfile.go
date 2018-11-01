package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/billgraziano/xelogstash/log"
	"github.com/pkg/errors"
)

func getLogFileName() (s string, err error) {
	s, err = os.Executable()
	if err != nil {
		return "", errors.Wrap(err, "os.executable")
	}
	s = filepath.Base(s)
	base := strings.TrimSuffix(s, path.Ext(s))
	s = base + "_" + time.Now().Format("20060102") + ".log"
	return s, nil
}

func cleanOldLogFiles(days int) error {
	var err error
	if days == 0 {
		return nil
	}

	log.Debug("cleanoldlogfiles.getlogfilename...")
	currentLog, err := getLogFileName()
	// it worked when I started.  It should work now.
	if err != nil {
		return nil
	}

	tz := time.Now().Location()
	hours := float64(days*24 + 1)

	exe, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "os.executable")
	}
	exe = filepath.Base(exe)
	exe = strings.TrimSuffix(exe, path.Ext(exe))

	log.Debug("cleanoldlogfiles.ioutil.readdir...")
	files, err := ioutil.ReadDir(".")
	if err != nil {
		return errors.Wrap(err, "readdir")
	}
	re := regexp.MustCompile(fmt.Sprintf("%s_(?P<date>\\d{8})\\.log", exe))
	for _, fi := range files {
		//log.Debug(fmt.Sprintf("ranging: %s", fi.Name()))
		if fi.IsDir() {
			continue
		}
		// don't delete the current log
		if fi.Name() == currentLog {
			continue
		}

		m := re.FindStringSubmatch(fi.Name())
		if len(m) == 0 {
			continue
		}
		if len(m) == 2 {
			fileDate := m[1]
			fd, err := time.ParseInLocation("20060102", fileDate, tz)
			if err != nil {
				continue
			}
			hoursago := time.Since(fd).Hours()
			if hoursago > hours {
				log.Debug("Deleting log file:", fi.Name())
				err = os.Remove(fi.Name())
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("os.remove: %s", fi.Name()))
				}
			}
		}
	}
	return nil
}
