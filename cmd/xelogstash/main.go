package main

import (
	"fmt"
	"os"

	"github.com/billgraziano/xelogstash/log"
)

const version = "0.32"

func main() {
	err := runApp()
	if err != nil {
		log.Error(fmt.Sprintf("runapp: %s", err.Error()))
		os.Exit(1)
	}
	log.Debug("exiting main")
}
