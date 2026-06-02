package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/flexfence/flexfence-backend/internal/domain"
	"github.com/flexfence/flexfence-backend/internal/store"
)

type EventConsentTemplateResponse struct {
	Configured     bool                   `json:"configured"`
	RequiredFields []domain.ConsentField  `json:"required_fields"`
	TrackEntryExit bool                 `json:"track_entry_exit"`
	TrackMovement  bool                   `json:"track_movement"`
}

type UserConsentStatusResponse struct {
	HasConsented bool                `json:"has_consented"`
	Consent      *domain.UserConsent `json:"consent,omitempty"`
}

type SubmitUserConsentRequest struct {
	Values map[string]string `json:"values"`
}

func myEventConsentHandler(dataStore store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := userAuthFromContext(r.Context())
		if !ok {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "User auth required")
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/v1/me/events/")
		path = strings.Trim(path, "/")
		parts := strings.Split(path, "/")
		if len(parts) < 2 {
			writeAPIError(w, http.StatusNotFound, "not_found", "Not found")
			return
		}
		eventID := parts[0]
		resource := parts[1]
		switch resource {
		case "consent-template":
			if r.Method != http.MethodGet {
				writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
				return
			}
			getMyEventConsentTemplateHandler(dataStore, user.UserID, eventID)(w, r)
		case "consent":
			switch r.Method {
			case http.MethodGet:
				getMyEventConsentStatusHandler(dataStore, user.UserID, eventID)(w, r)
			case http.MethodPost:
				submitMyEventConsentHandler(dataStore, user.UserID, eventID)(w, r)
			default:
				writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			}
		default:
			writeAPIError(w, http.StatusNotFound, "not_found", "Not found")
		}
	}
}

func getMyEventConsentTemplateHandler(dataStore store.Store, userID, eventID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		joined, err := dataStore.UserHasJoinedEvent(eventID, userID)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		if !joined {
			writeAPIError(w, http.StatusForbidden, "not_joined_event", "You must join the event before viewing consent")
			return
		}
		tpl, ok, err := dataStore.GetConsentTemplate(eventID)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		if !ok || len(tpl.RequiredFields) == 0 {
			writeJSON(w, http.StatusOK, EventConsentTemplateResponse{Configured: false, RequiredFields: []domain.ConsentField{}})
			return
		}
		writeJSON(w, http.StatusOK, EventConsentTemplateResponse{
			Configured:     true,
			RequiredFields: tpl.RequiredFields,
			TrackEntryExit: tpl.TrackEntryExit,
			TrackMovement:  tpl.TrackMovement,
		})
	}
}

func getMyEventConsentStatusHandler(dataStore store.Store, userID, eventID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		joined, err := dataStore.UserHasJoinedEvent(eventID, userID)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		if !joined {
			writeAPIError(w, http.StatusForbidden, "not_joined_event", "You must join the event before viewing consent")
			return
		}
		consent, ok, err := dataStore.GetUserEventConsent(eventID, userID)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		resp := UserConsentStatusResponse{HasConsented: ok}
		if ok {
			resp.Consent = &consent
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func submitMyEventConsentHandler(dataStore store.Store, userID, eventID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		joined, err := dataStore.UserHasJoinedEvent(eventID, userID)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		if !joined {
			writeAPIError(w, http.StatusForbidden, "not_joined_event", "You must join the event before submitting consent")
			return
		}
		tpl, ok, err := dataStore.GetConsentTemplate(eventID)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		if !ok || len(tpl.RequiredFields) == 0 {
			writeAPIError(w, http.StatusBadRequest, "consent_not_configured", "This event does not require consent")
			return
		}
		var req SubmitUserConsentRequest
		if err := decodeJSON(r, &req); err != nil {
			writeInvalidJSON(w, err)
			return
		}
		if req.Values == nil {
			req.Values = map[string]string{}
		}
		consent, err := dataStore.SaveUserEventConsent(eventID, userID, req.Values, tpl)
		if err != nil {
			if errors.Is(err, store.ErrInvalidConsent) {
				writeStoreErr(w, err)
				return
			}
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		writeJSON(w, http.StatusCreated, consent)
	}
}

func validateClockInAccess(dataStore store.Store, userID, eventID, source string) error {
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return nil
	}
	joined, err := dataStore.UserHasJoinedEvent(eventID, userID)
	if err != nil {
		return err
	}
	if !joined {
		return store.ErrNotJoinedEvent
	}
	scanRequired, err := dataStore.EventScanToClockInEnabled(eventID)
	if err != nil {
		return err
	}
	if scanRequired && strings.TrimSpace(source) != "qr_scan" {
		return store.ErrQRScanRequired
	}
	tpl, ok, err := dataStore.GetConsentTemplate(eventID)
	if err != nil {
		return err
	}
	if ok && len(tpl.RequiredFields) > 0 {
		has, err := dataStore.UserHasEventConsent(eventID, userID)
		if err != nil {
			return err
		}
		if !has {
			return store.ErrConsentRequired
		}
	}
	return nil
}
