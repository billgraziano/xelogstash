package main

import (
	"sync"

	"github.com/billgraziano/xelogstash/app"
	"github.com/billgraziano/xelogstash/config"
)

func worker(id int, wg *sync.WaitGroup, jobs <-chan config.Source, results chan<- int, errors chan<- error) {
	//	count := 0
	sinks, err := globalConfig.GetSinks()
	if err != nil {
		errors <- err
		wg.Done()
		return
	}

	for j := range jobs {
		result, err := app.ProcessSource(id, j, sinks)
		results <- result.Rows
		if err != nil {
			errors <- err
		}
		//		count++
		wg.Done()
	}
}
