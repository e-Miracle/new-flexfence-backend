package http

import (
	"net/http"

	"github.com/flexfence/flexfence-backend/internal/store"
)

func getEventAnalyticsHandler(dataStore store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		eventID := eventIDFromPath(r.URL.Path)
		analytics, err := dataStore.GetEventAnalytics(eventID)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, analytics)
	}
}

func getFenceAnalyticsHandler(dataStore store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		eventID := eventIDFromPath(r.URL.Path)
		fenceID := fenceIDFromPath(r.URL.Path)
		analytics, err := dataStore.GetFenceAnalytics(eventID, fenceID)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, analytics)
	}
}
