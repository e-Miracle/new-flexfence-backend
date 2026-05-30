package http

import (
	"net/http"

	"github.com/flexfence/flexfence-backend/internal/store"
)

func getEventShareHandler(dataStore store.Store, joinPublicBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		biz, ok := businessAuthFromContext(r.Context())
		if !ok {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
			return
		}
		eventID := eventIDFromPath(r.URL.Path)
		event, found, err := dataStore.GetEventForOrganization(eventID, biz.OrganizationID)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		if !found {
			writeAPIError(w, http.StatusNotFound, "event_not_found", "Event was not found")
			return
		}

		qrToken, err := dataStore.EnsureEventQRToken(eventID)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, buildEventShare(event, qrToken, joinPublicBase))
	}
}

func regenerateEventQRHandler(dataStore store.Store, joinPublicBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		biz, ok := businessAuthFromContext(r.Context())
		if !ok {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
			return
		}
		eventID := eventIDFromPath(r.URL.Path)
		event, found, err := dataStore.GetEventForOrganization(eventID, biz.OrganizationID)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		if !found {
			writeAPIError(w, http.StatusNotFound, "event_not_found", "Event was not found")
			return
		}

		qrToken, err := dataStore.RegenerateEventQRToken(eventID)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, buildEventShare(event, qrToken, joinPublicBase))
	}
}
