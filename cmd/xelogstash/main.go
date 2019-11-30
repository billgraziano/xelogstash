package main

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
)

var (
	sha1ver   = "dev"
	version   = "dev"
	buildTime string
)

func main() {
	err := runApp()

	if err != nil {
		log.Error(fmt.Sprintf("runapp: %s", err.Error()))
		os.Exit(1)
	}
	log.Debug("exiting main")
}
