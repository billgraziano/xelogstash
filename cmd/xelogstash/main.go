package main

import (
	"os"
)

const version = "0.28"

func main() {
	err := runApp()
	if err != nil {
		os.Exit(1)
	}
}
