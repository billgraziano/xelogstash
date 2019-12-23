package app

import (
	"context"
	"fmt"
	"runtime"
	"time"

	log "github.com/sirupsen/logrus"
)

func (p *Program) logMemory(ctx context.Context, count int) {
	// context sleep for 60 seconds
	sleep(ctx, 60*time.Second)
	if ctx.Err() != nil {
		return
	}
	writeMemory(p.StartTime, count)

	go func(ctx context.Context) {
		ticker := time.NewTicker(24 * time.Hour)
		for {
			select {
			case <-ticker.C:
				writeMemory(p.StartTime, count)
			case <-ctx.Done():
				return
			}
		}
	}(ctx)
}

func writeMemory(start time.Time, count int) {
	m := &runtime.MemStats{}
	runtime.ReadMemStats(m)
	msg := fmt.Sprintf("metrics: alloc: %.1fmb; sys: %.1fmb; goroutines: %d; uptime: %s; sources: %d",
		float64(m.Alloc)/(1024.0*1024.0), float64(m.Sys)/(1024.0*1024.0), runtime.NumGoroutine(), time.Since(start), count)
	log.Info(msg)
}
