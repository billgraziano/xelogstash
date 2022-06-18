package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/billgraziano/xelogstash/pkg/config"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

/*

/app/program.go/startPolling()
	/app/program_config.go/getConfig()
		/app/program_config.go/getConfigFiles() - file names
		/config/config.go/Get() -- reads files and build configuration
			/config.decodekv()

*/

func (p *Program) getConfig() (config.Config, error) {
	var c config.Config
	var err error

	// get the dir of the EXE

	cfg, src, err := getConfigFiles()
	if err != nil {
		return c, errors.Wrap(err, "getconfigfiles")
	}
	c, err = config.Get(cfg, src, p.Version, p.SHA1)
	if err != nil {
		return c, errors.Wrap(err, "config.get")
	}
	p.Filters = c.Filters
	return c, nil
}

func getConfigFiles() (cfg string, src string, err error) {

	// get the dir of the EXE
	// c:\dir\sqlxewriter.exe
	exe, err := os.Executable()
	if err != nil {
		return "", "", errors.Wrap(err, "os.executable")
	}
	exePath := filepath.Dir(exe)
	base := filepath.Base(exe)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	config := filepath.Join(exePath, base+".toml")
	log.Debugf("desired config file: %s", config)

	// Does the desired config file exist
	_, err = os.Stat(config)
	if os.IsNotExist(err) {
		return "", "", fmt.Errorf("missing config file: %s", config)
	}
	if err != nil {
		return "", "", errors.Wrap(err, "os.stat")
	}
	cfg = config

	// Does the sources file exist
	sources := filepath.Join(exePath, base+"_sources.toml")
	log.Debugf("checking sources file: %s", sources)
	_, err = os.Stat(sources)
	if os.IsNotExist(err) {
		return cfg, "", nil
	}
	if err != nil {
		return "", "", errors.Wrap(err, "os.state: sources")
	}
	src = sources
	return cfg, src, nil
}
