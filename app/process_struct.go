package app

import "github.com/billgraziano/xelogstash/config"

// Result holds the result from a source or session process
type Result struct {
	Source   config.Source
	Instance string
	Session  string
	Rows     int
	//Error    error
	//Duration time.Duration
}
