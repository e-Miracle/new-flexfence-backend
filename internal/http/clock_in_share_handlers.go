package http

import (
	"net/http"

	"github.com/flexfence/flexfence-backend/internal/store"
)

type UpdateEventClockInSettingsRequest struct {
	ScanToClockInEnabled     bool `json:"scan_to_clock_in_enabled"`
	ClockInQRRotationMinutes int  `json:"clock_in_qr_rotation_minutes"`
}

func getEventClockInShareHandler(dataStore store.Store, joinPublicBase string) http.HandlerFunc {
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
		if !event.ScanToClockInEnabled {
			writeJSON(w, http.StatusOK, buildEventClockInShare(event, "", event.CreatedAt, joinPublicBase))
			return
		}
		token, issuedAt, err := dataStore.EnsureEventClockInQRToken(eventID)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, buildEventClockInShare(event, token, issuedAt, joinPublicBase))
	}
}

func patchEventClockInSettingsHandler(dataStore store.Store, joinPublicBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
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

		var req UpdateEventClockInSettingsRequest
		if err := decodeJSON(r, &req); err != nil {
			writeInvalidJSON(w, err)
			return
		}
		if err := dataStore.UpdateEventClockInSettings(eventID, req.ScanToClockInEnabled, req.ClockInQRRotationMinutes); err != nil {
			writeStoreErr(w, err)
			return
		}

		event, found, err = dataStore.GetEventForOrganization(eventID, biz.OrganizationID)
		if err != nil || !found {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		if !event.ScanToClockInEnabled {
			writeJSON(w, http.StatusOK, buildEventClockInShare(event, "", event.CreatedAt, joinPublicBase))
			return
		}
		token, issuedAt, err := dataStore.EnsureEventClockInQRToken(eventID)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, buildEventClockInShare(event, token, issuedAt, joinPublicBase))
	}
}

func regenerateEventClockInShareHandler(dataStore store.Store, joinPublicBase string) http.HandlerFunc {
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
		if !event.ScanToClockInEnabled {
			writeAPIError(w, http.StatusBadRequest, "clock_in_qr_disabled", "Scan to clock-in is not enabled for this event")
			return
		}

		token, issuedAt, err := dataStore.RegenerateEventClockInQRToken(eventID)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, buildEventClockInShare(event, token, issuedAt, joinPublicBase))
	}
}
