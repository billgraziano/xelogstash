package main

import (
	"sync"

	"github.com/billgraziano/xelogstash/config"
)

// Result holds the result from a source or session process
type Result struct {
	Source   config.Source
	Instance string
	Session  string
	Rows     int
	//Error    error
	//Duration time.Duration
}

func worker(id int, wg *sync.WaitGroup, jobs <-chan config.Source, results chan<- int, errors chan<- error) {
	//	count := 0
	for j := range jobs {
		result, err := processSource(id, j)
		results <- result.Rows
		if err != nil {
			errors <- err
		}
		//		count++
		wg.Done()
	}
}
