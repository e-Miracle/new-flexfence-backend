package http

import (
	"net/http"
	"strings"

	"github.com/flexfence/flexfence-backend/internal/domain"
	"github.com/flexfence/flexfence-backend/internal/store"
)

func updateEventHandler(dataStore store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		biz, ok := businessAuthFromContext(r.Context())
		if !ok {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Business auth required")
			return
		}
		eventID := eventIDFromPath(r.URL.Path)
		var req CreateEventRequest
		if err := decodeJSON(r, &req); err != nil {
			writeInvalidJSON(w, err)
			return
		}
		if strings.TrimSpace(req.Title) == "" {
			writeAPIError(w, http.StatusBadRequest, "title_required", "title is required")
			return
		}
		startAt, err := parseRFC3339Time(req.StartAt, "start_at")
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_schedule", err.Error())
			return
		}
		endAt, err := parseRFC3339Time(req.EndAt, "end_at")
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_schedule", err.Error())
			return
		}
		if err := domain.ValidateEventSchedule(startAt, endAt); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_schedule", err.Error())
			return
		}
		event, err := dataStore.UpdateEvent(
			eventID,
			biz.OrganizationID,
			req.Title,
			req.Description,
			startAt,
			endAt,
			req.GeofenceGpsTolerance,
		)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, event)
	}
}

func deleteEventHandler(dataStore store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		biz, ok := businessAuthFromContext(r.Context())
		if !ok {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Business auth required")
			return
		}
		eventID := eventIDFromPath(r.URL.Path)
		if err := dataStore.DeleteEvent(eventID, biz.OrganizationID); err != nil {
			writeStoreErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
