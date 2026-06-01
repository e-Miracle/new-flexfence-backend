package http

import (
	"net/http"
	"strings"

	"github.com/flexfence/flexfence-backend/internal/domain"
	"github.com/flexfence/flexfence-backend/internal/store"
)

func getMyNotificationPreferencesHandler(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := userAuthFromContext(r.Context())
		if !ok {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "User auth required")
			return
		}
		prefs, err := deps.IdentityStore.GetUserNotificationPreferences(user.UserID)
		if err != nil {
			if err == store.ErrUserNotFound {
				writeAPIError(w, http.StatusNotFound, "user_not_found", "User account not found")
				return
			}
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		writeJSON(w, http.StatusOK, prefs)
	}
}

func updateMyNotificationPreferencesHandler(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := userAuthFromContext(r.Context())
		if !ok {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "User auth required")
			return
		}
		var req domain.UserNotificationPreferences
		if err := decodeJSON(r, &req); err != nil {
			writeInvalidJSON(w, err)
			return
		}
		updated, err := deps.IdentityStore.UpdateUserNotificationPreferences(user.UserID, req)
		if err != nil {
			if err == store.ErrUserNotFound {
				writeAPIError(w, http.StatusNotFound, "user_not_found", "User account not found")
				return
			}
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		writeJSON(w, http.StatusOK, updated)
	}
}

type RegisterDeviceTokenRequest struct {
	Platform string `json:"platform"`
	Token    string `json:"token"`
}

func registerMyDeviceTokenHandler(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := userAuthFromContext(r.Context())
		if !ok {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "User auth required")
			return
		}
		var req RegisterDeviceTokenRequest
		if err := decodeJSON(r, &req); err != nil {
			writeInvalidJSON(w, err)
			return
		}
		if err := deps.IdentityStore.UpsertUserDeviceToken(user.UserID, req.Platform, req.Token); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_token", err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func deleteMyDeviceTokenHandler(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := userAuthFromContext(r.Context())
		if !ok {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "User auth required")
			return
		}
		var req RegisterDeviceTokenRequest
		if err := decodeJSON(r, &req); err != nil {
			writeInvalidJSON(w, err)
			return
		}
		_ = deps.IdentityStore.DeleteUserDeviceToken(user.UserID, req.Token)
		w.WriteHeader(http.StatusNoContent)
	}
}

type DispatchNotificationRequest struct {
	Type       string `json:"type"`
	EventID    string `json:"event_id,omitempty"`
	EventTitle string `json:"event_title,omitempty"`
	FenceID    string `json:"fence_id,omitempty"`
	FenceName  string `json:"fence_name,omitempty"`
	Message    string `json:"message"`
}

func dispatchMyNotificationHandler(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := userAuthFromContext(r.Context())
		if !ok {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "User auth required")
			return
		}
		if deps.Notifier == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		var req DispatchNotificationRequest
		if err := decodeJSON(r, &req); err != nil {
			writeInvalidJSON(w, err)
			return
		}
		alertType := strings.TrimSpace(req.Type)
		if alertType == "" {
			writeAPIError(w, http.StatusBadRequest, "type_required", "type is required")
			return
		}
		alert := domain.GeofenceAlert{
			Type:       alertType,
			EventID:    strings.TrimSpace(req.EventID),
			EventTitle: strings.TrimSpace(req.EventTitle),
			FenceID:    strings.TrimSpace(req.FenceID),
			FenceName:  strings.TrimSpace(req.FenceName),
			Message:    strings.TrimSpace(req.Message),
		}
		deps.Notifier.DispatchGeofenceAlert(r.Context(), user.UserID, alert)
		w.WriteHeader(http.StatusNoContent)
	}
}
