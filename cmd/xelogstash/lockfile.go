package main

import (
	"os"
	"time"

	"github.com/billgraziano/xelogstash/log"
	"github.com/pkg/errors"
)

func closeLockFile(f *os.File) error {
	ch := make(chan bool, 1)
	defer close(ch)

	go func() {
		err := f.Close()
		if err != nil {
			msg := errors.Wrap(err, "os.close").Error()
			log.Error(msg)
		}
		ch <- true
	}()
	select {
	case <-ch:

	case <-time.After(31 * time.Second):
		err := errors.New("timeout")
		return err
	}
	return nil
}

func removeLockFile(fn string) error {
	ch := make(chan bool, 1)
	defer close(ch)

	go func() {
		err := os.Remove(fn)
		if err != nil {
			msg := errors.Wrap(err, "os.remove").Error()
			log.Error(msg)
		}
		ch <- true
	}()
	select {
	case <-ch:

	case <-time.After(31 * time.Second):
		err := errors.New("timeout")
		return err
	}

	return nil
}
