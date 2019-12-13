package app

import (
	"context"
	"fmt"
	"runtime"
	"time"

	log "github.com/sirupsen/logrus"
)

func logMemory(ctx context.Context) {
	now := time.Now()
	go func(start time.Time, ctx context.Context) {
		ticker := time.NewTicker(24 * time.Hour)
		for {
			select {
			case <-ticker.C:
				m := &runtime.MemStats{}
				runtime.ReadMemStats(m)
				msg := fmt.Sprintf("alloc: %.1fmb; sys: %.1fmb; goroutines: %d; running: %s",
					float64(m.Alloc)/(1024.0*1024.0), float64(m.Sys)/(1024.0*1024.0), runtime.NumGoroutine(), time.Since(start))
				log.Info(msg)
			case <-ctx.Done():
				return
			}
		}
	}(now, ctx)
}
