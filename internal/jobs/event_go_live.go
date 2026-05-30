package jobs

import (
	"log"
	"time"

	"github.com/flexfence/flexfence-backend/internal/store"
)

// StartEventGoLiveMonitor runs a background job that processes events entering their live window.
func StartEventGoLiveMonitor(dataStore store.Store, interval time.Duration) {
	if interval <= 0 {
		interval = time.Minute
	}
	go func() {
		process := func() {
			count, err := dataStore.ProcessPendingEventGoLive(time.Now().UTC())
			if err != nil {
				log.Printf("event go-live monitor failed: %v", err)
				return
			}
			if count > 0 {
				log.Printf("event go-live monitor: processed %d event(s)", count)
			}
		}
		process()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			process()
		}
	}()
}
