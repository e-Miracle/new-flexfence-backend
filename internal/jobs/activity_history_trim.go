package jobs

import (
	"log"
	"time"

	"github.com/flexfence/flexfence-backend/internal/store"
)

const ActivityHistoryRetentionDays = 7

// StartActivityHistoryTrim runs a background job that deletes activity history older than the retention window.
func StartActivityHistoryTrim(dataStore store.Store, interval time.Duration) {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	go func() {
		trim := func() {
			cutoff := time.Now().UTC().AddDate(0, 0, -ActivityHistoryRetentionDays)
			deleted, err := dataStore.DeleteActivityHistoryOlderThan(cutoff)
			if err != nil {
				log.Printf("activity history trim failed: %v", err)
				return
			}
			if deleted > 0 {
				log.Printf("activity history trim: deleted %d session(s) older than %d days", deleted, ActivityHistoryRetentionDays)
			}
		}
		trim()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			trim()
		}
	}()
}
