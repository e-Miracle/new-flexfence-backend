package http

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/flexfence/flexfence-backend/internal/auth"
	"github.com/flexfence/flexfence-backend/internal/domain"
	"github.com/flexfence/flexfence-backend/internal/jobs"
	"github.com/flexfence/flexfence-backend/internal/store"
	mysqlstore "github.com/flexfence/flexfence-backend/internal/store/mysql"
)

func listMyEventJoinsHandler(dataStore store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		user, ok := userAuthFromContext(r.Context())
		if !ok {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "User auth required")
			return
		}

		filter := parseUserEventJoinFilter(r)
		page, err := dataStore.ListJoinsByUserFiltered(user.UserID, filter)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		if page.Joins == nil {
			page.Joins = []domain.UserEventJoin{}
		}
		writeJSON(w, http.StatusOK, ListUserEventJoinsResponse{
			Joins:   page.Joins,
			Total:   page.Total,
			Page:    page.Page,
			Limit:   page.Limit,
			HasMore: page.HasMore,
		})
	}
}

func listMySubscribedEventsHandler(dataStore store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		user, ok := userAuthFromContext(r.Context())
		if !ok {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "User auth required")
			return
		}
		events, err := dataStore.ListSubscribedGeofenceEvents(user.UserID, time.Now().UTC())
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		alerts, err := dataStore.ConsumeUserGeofenceAlerts(user.UserID)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		if events == nil {
			events = []domain.SubscribedGeofenceEvent{}
		}
		if alerts == nil {
			alerts = []domain.GeofenceAlert{}
		}
		for i := range events {
			if events[i].Fences == nil {
				events[i].Fences = []domain.Fence{}
			}
		}
		writeJSON(w, http.StatusOK, ListSubscribedGeofenceEventsResponse{
			Events:      events,
			Alerts:      alerts,
			RefreshedAt: time.Now().UTC().Format(time.RFC3339),
		})
	}
}

func deleteMyEventJoinHandler(dataStore store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		user, ok := userAuthFromContext(r.Context())
		if !ok {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "User auth required")
			return
		}
		joinID := strings.TrimPrefix(r.URL.Path, "/v1/me/event-joins/")
		joinID = strings.Trim(joinID, "/")
		if joinID == "" {
			writeAPIError(w, http.StatusBadRequest, "join_id_required", "Join id is required")
			return
		}
		if err := dataStore.DeleteUserEventJoin(user.UserID, joinID); err != nil {
			if err == store.ErrJoinNotFound {
				writeAPIError(w, http.StatusNotFound, "join_not_found", "Event join not found")
				return
			}
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func parseUserEventJoinFilter(r *http.Request) store.UserEventJoinFilter {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	filter := store.UserEventJoinFilter{
		Search:     strings.TrimSpace(q.Get("q")),
		Status:     strings.TrimSpace(q.Get("status")),
		JoinSource: strings.TrimSpace(q.Get("join_source")),
		Page:       page,
		Limit:      limit,
	}
	if raw := strings.TrimSpace(q.Get("start_from")); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			filter.StartFrom = &t
		}
	}
	if raw := strings.TrimSpace(q.Get("start_to")); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			filter.StartTo = &t
		}
	}
	return filter
}

func listMyActivityHistoryHandler(dataStore store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		user, ok := userAuthFromContext(r.Context())
		if !ok {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "User auth required")
			return
		}
		filter := parseActivityHistoryFilter(r)
		sessions, err := dataStore.ListUserActivityHistory(user.UserID, filter)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		if sessions == nil {
			sessions = []domain.UserActivitySession{}
		}
		writeJSON(w, http.StatusOK, ListUserActivityHistoryResponse{
			Sessions:      sessions,
			RetentionDays: jobs.ActivityHistoryRetentionDays,
		})
	}
}

func recordClockInHandler(dataStore store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		user, ok := userAuthFromContext(r.Context())
		if !ok {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "User auth required")
			return
		}
		var req RecordClockInRequest
		if err := decodeJSON(r, &req); err != nil {
			writeInvalidJSON(w, err)
			return
		}
		session, err := dataStore.RecordClockIn(
			user.UserID,
			req.EventID,
			req.EventTitle,
			req.FenceID,
			req.FenceName,
			req.Source,
			req.Lat,
			req.Lng,
		)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		writeJSON(w, http.StatusCreated, session)
	}
}

func recordClockOutHandler(dataStore store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		user, ok := userAuthFromContext(r.Context())
		if !ok {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "User auth required")
			return
		}
		var req RecordClockOutRequest
		if err := decodeJSON(r, &req); err != nil {
			writeInvalidJSON(w, err)
			return
		}
		session, err := dataStore.RecordClockOut(user.UserID, req.SessionID, req.EventID, req.FenceID)
		if err != nil {
			if err == store.ErrOpenSessionNotFound {
				writeAPIError(w, http.StatusNotFound, "open_session_not_found", "No open clock-in session was found")
				return
			}
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		writeJSON(w, http.StatusOK, session)
	}
}

func parseActivityHistoryFilter(r *http.Request) store.ActivityHistoryFilter {
	q := r.URL.Query()
	filter := store.ActivityHistoryFilter{
		Period: strings.TrimSpace(q.Get("period")),
	}
	if raw := strings.TrimSpace(q.Get("from")); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			filter.From = &t
		}
	}
	if raw := strings.TrimSpace(q.Get("to")); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			filter.To = &t
		}
	}
	return filter
}

func updateMyProfileHandler(identityStore *mysqlstore.IdentityStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		user, ok := userAuthFromContext(r.Context())
		if !ok {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "User auth required")
			return
		}
		if identityStore == nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "Profile update is unavailable")
			return
		}
		var req UpdateMyProfileRequest
		if err := decodeJSON(r, &req); err != nil {
			writeInvalidJSON(w, err)
			return
		}
		firstName := strings.TrimSpace(req.FirstName)
		if firstName == "" {
			writeAPIError(w, http.StatusBadRequest, "first_name_required", "first_name is required")
			return
		}
		updated, err := identityStore.UpdateUserProfile(user.UserID, firstName, strings.TrimSpace(req.LastName))
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

func deleteMyAccountHandler(identityStore *mysqlstore.IdentityStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		user, ok := userAuthFromContext(r.Context())
		if !ok {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "User auth required")
			return
		}
		if identityStore == nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "Account deletion is unavailable")
			return
		}
		if err := identityStore.DeleteAttendeeUser(user.UserID); err != nil {
			if err == store.ErrUserNotFound {
				writeAPIError(w, http.StatusNotFound, "user_not_found", "User account not found")
				return
			}
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func changePasswordHandler(identityStore *mysqlstore.IdentityStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		user, ok := userAuthFromContext(r.Context())
		if !ok {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "User auth required")
			return
		}
		if identityStore == nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "Password change is unavailable")
			return
		}
		var req ChangePasswordRequest
		if err := decodeJSON(r, &req); err != nil {
			writeInvalidJSON(w, err)
			return
		}
		current := strings.TrimSpace(req.CurrentPassword)
		newPassword := strings.TrimSpace(req.NewPassword)
		if current == "" || newPassword == "" {
			writeAPIError(w, http.StatusBadRequest, "password_required", "current_password and new_password are required")
			return
		}
		if pwdErr := auth.ValidatePassword(newPassword); pwdErr != nil {
			writeAPIError(w, http.StatusBadRequest, pwdErr.Code, pwdErr.Message)
			return
		}
		record, found, err := identityStore.GetUserAuthByID(user.UserID)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		if !found {
			writeAPIError(w, http.StatusNotFound, "user_not_found", "User account not found")
			return
		}
		if record.PasswordHash == "" {
			writeAPIError(w, http.StatusBadRequest, "password_not_set", "No password is set for this account")
			return
		}
		if !auth.CheckPassword(record.PasswordHash, current) {
			writeAPIError(w, http.StatusUnauthorized, "invalid_current_password", "Current password is incorrect")
			return
		}
		if auth.CheckPassword(record.PasswordHash, newPassword) {
			writeAPIError(w, http.StatusBadRequest, "password_unchanged", "New password must be different from the current password")
			return
		}
		hash, err := auth.HashPassword(newPassword)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		if err := identityStore.UpdateUserPasswordHash(user.UserID, hash); err != nil {
			if err == store.ErrUserNotFound {
				writeAPIError(w, http.StatusNotFound, "user_not_found", "User account not found")
				return
			}
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
