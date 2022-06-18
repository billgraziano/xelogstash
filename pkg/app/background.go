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
	select {
	case <-ctx.Done():
		// write memory on exit
		writeMemory(p.StartTime, count)
		return
	case <-time.After(time.Duration(60 * time.Second)):
		break
	}
	writeMemory(p.StartTime, count)

	go func(ctx context.Context) {
		ticker := time.NewTicker(24 * time.Hour)
		for {
			select {
			case <-ticker.C:
				writeMemory(p.StartTime, count)
			case <-ctx.Done():
				writeMemory(p.StartTime, count)
				return
			}
		}
	}(ctx)
}

func writeMemory(start time.Time, count int) {
	m := &runtime.MemStats{}
	runtime.ReadMemStats(m)
	msg := fmt.Sprintf("metrics: alloc: %.1fmb; sys: %.1fmb; goroutines: %d; uptime: %s; sources: %d",
		float64(m.Alloc)/(1024.0*1024.0), float64(m.Sys)/(1024.0*1024.0), runtime.NumGoroutine(), fmtduration(time.Since(start)), count)
	log.Info(msg)
}

// func sleep(ctx context.Context, dur time.Duration) {
// 	select {
// 	case <-ctx.Done():
// 		return
// 	case <-time.After(dur):
// 		break
// 	}
// }
