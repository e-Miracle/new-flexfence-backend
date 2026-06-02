package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/flexfence/flexfence-backend/internal/domain"
	"github.com/flexfence/flexfence-backend/internal/store"
	mysqlstore "github.com/flexfence/flexfence-backend/internal/store/mysql"
)

func NewRouter(deps RouterDeps) http.Handler {
	g := NewGuards(deps.Tokens, deps.IdentityStore)
	mux := http.NewServeMux()

	mux.Handle("/health", g.Public(http.HandlerFunc(healthHandler)))
	mux.Handle("/v1/public/legal/privacy", g.Public(http.HandlerFunc(publicPrivacyPolicyHandler)))
	mux.Handle("/v1/public/legal/terms", g.Public(http.HandlerFunc(publicTermsOfServiceHandler)))
	mux.Handle("/v1/auth/business/register", g.Public(http.HandlerFunc(businessRegisterHandler(deps))))
	mux.Handle("/v1/auth/business/login", g.Public(http.HandlerFunc(businessLoginHandler(deps))))
	mux.Handle("/v1/auth/business/otp/verify", g.Public(http.HandlerFunc(businessOTPVerifyHandler(deps))))
	mux.Handle("/v1/auth/business/otp/resend", g.Public(http.HandlerFunc(businessOTPResendHandler(deps))))
	mux.Handle("/v1/auth/business/oauth/google", g.Public(http.HandlerFunc(businessGoogleOAuthHandler(deps))))
	mux.Handle("/v1/auth/user/register", g.Public(http.HandlerFunc(userRegisterHandler(deps))))
	mux.Handle("/v1/auth/user/login", g.Public(http.HandlerFunc(userLoginHandler(deps))))
	mux.Handle("/v1/auth/user/otp/verify", g.Public(http.HandlerFunc(userOTPVerifyHandler(deps))))
	mux.Handle("/v1/auth/user/otp/resend", g.Public(http.HandlerFunc(userOTPResendHandler(deps))))
	mux.Handle("/v1/auth/user/oauth/google", g.Public(http.HandlerFunc(userGoogleOAuthHandler(deps))))
	mux.Handle("/v1/auth/user/password-reset/request", g.Public(http.HandlerFunc(userPasswordResetRequestHandler(deps))))
	mux.Handle("/v1/auth/user/password-reset/resend", g.Public(http.HandlerFunc(userPasswordResetResendHandler(deps))))
	mux.Handle("/v1/auth/user/password-reset/confirm", g.Public(http.HandlerFunc(userPasswordResetConfirmHandler(deps))))

	mux.Handle("/v1/me/notification-preferences", Chain(
		g.AllowMethods(http.MethodGet, http.MethodPatch),
		func(h http.Handler) http.Handler { return g.User(h) },
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getMyNotificationPreferencesHandler(deps)(w, r)
		case http.MethodPatch:
			updateMyNotificationPreferencesHandler(deps)(w, r)
		default:
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		}
	})))

	mux.Handle("/v1/me/device-tokens", Chain(
		g.AllowMethods(http.MethodPost, http.MethodDelete),
		func(h http.Handler) http.Handler { return g.User(h) },
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			registerMyDeviceTokenHandler(deps)(w, r)
		case http.MethodDelete:
			deleteMyDeviceTokenHandler(deps)(w, r)
		default:
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		}
	})))

	mux.Handle("/v1/me/notifications/dispatch", Chain(
		g.AllowMethods(http.MethodPost),
		func(h http.Handler) http.Handler { return g.User(h) },
	)(http.HandlerFunc(dispatchMyNotificationHandler(deps))))

	mux.Handle("/v1/me/event-joins", Chain(
		g.AllowMethods(http.MethodGet),
		func(h http.Handler) http.Handler { return g.User(h) },
	)(http.HandlerFunc(listMyEventJoinsHandler(deps.DataStore))))

	mux.Handle("/v1/me/subscribed-events", Chain(
		g.AllowMethods(http.MethodGet),
		func(h http.Handler) http.Handler { return g.User(h) },
	)(http.HandlerFunc(listMySubscribedEventsHandler(deps.DataStore))))

	mux.Handle("/v1/me/event-joins/", Chain(
		g.AllowMethods(http.MethodDelete),
		func(h http.Handler) http.Handler { return g.User(h) },
	)(http.HandlerFunc(deleteMyEventJoinHandler(deps.DataStore))))

	mux.Handle("/v1/me/events/", Chain(
		func(h http.Handler) http.Handler { return g.User(h) },
	)(http.HandlerFunc(myEventConsentHandler(deps.DataStore))))

	mux.Handle("/v1/me", Chain(
		g.AllowMethods(http.MethodDelete, http.MethodPatch),
		func(h http.Handler) http.Handler { return g.User(h) },
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodDelete:
			deleteMyAccountHandler(deps.IdentityStore)(w, r)
		case http.MethodPatch:
			updateMyProfileHandler(deps.IdentityStore)(w, r)
		default:
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		}
	})))

	mux.Handle("/v1/me/change-password", Chain(
		g.AllowMethods(http.MethodPost),
		func(h http.Handler) http.Handler { return g.User(h) },
	)(http.HandlerFunc(changePasswordHandler(deps.IdentityStore))))

	mux.Handle("/v1/me/activity-history", Chain(
		g.AllowMethods(http.MethodGet),
		func(h http.Handler) http.Handler { return g.User(h) },
	)(http.HandlerFunc(listMyActivityHistoryHandler(deps.DataStore))))

	mux.Handle("/v1/me/activity-history/clock-in", Chain(
		g.AllowMethods(http.MethodPost),
		func(h http.Handler) http.Handler { return g.User(h) },
	)(http.HandlerFunc(recordClockInHandler(deps.DataStore))))

	mux.Handle("/v1/me/activity-history/clock-out", Chain(
		g.AllowMethods(http.MethodPost),
		func(h http.Handler) http.Handler { return g.User(h) },
	)(http.HandlerFunc(recordClockOutHandler(deps.DataStore))))

	mux.Handle("/v1/public/fence-capture/", g.Public(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/points") {
			publicSubmitCapturePointHandler(deps.DataStore)(w, r)
			return
		}
		publicGetFenceCaptureHandler(deps.DataStore)(w, r)
	})))

	mux.Handle("/v1/consent-field-recommendations", Chain(
		g.AllowMethods(http.MethodGet),
		func(h http.Handler) http.Handler { return g.BusinessRead(h) },
	)(http.HandlerFunc(listConsentFieldRecommendationsHandler(deps.DataStore))))

	mux.Handle("/v1/events", Chain(
		g.AllowMethods(http.MethodGet, http.MethodPost),
		func(h http.Handler) http.Handler { return g.BusinessRead(h) },
	)(http.HandlerFunc(dashboardEventsHandler(deps.DataStore))))

	mux.Handle("/v1/events/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch classifyEventRoute(r) {
		case eventRouteJoinQR:
			Chain(
				g.AllowMethods(http.MethodPost),
				func(h http.Handler) http.Handler { return g.OptionalUser(h) },
			)(http.HandlerFunc(mobileEventHandler(deps.DataStore, deps.IdentityStore))).ServeHTTP(w, r)
		case eventRouteMarkPresent:
			Chain(
				g.AllowMethods(http.MethodPost),
				func(h http.Handler) http.Handler { return g.User(h) },
			)(http.HandlerFunc(mobileEventHandler(deps.DataStore, deps.IdentityStore))).ServeHTTP(w, r)
		case eventRouteGet:
			Chain(
				g.AllowMethods(http.MethodGet),
				func(h http.Handler) http.Handler { return g.BusinessRead(h) },
				g.RequireEventTenant(deps.DataStore),
			)(http.HandlerFunc(dashboardEventHandler(deps.DataStore))).ServeHTTP(w, r)
		case eventRouteFence:
			Chain(
				g.AllowMethods(http.MethodPost),
				func(h http.Handler) http.Handler { return g.BusinessRead(h) },
				g.RequireEventTenant(deps.DataStore),
			)(http.HandlerFunc(dashboardEventHandler(deps.DataStore))).ServeHTTP(w, r)
		case eventRouteFencesList:
			Chain(
				g.AllowMethods(http.MethodGet),
				func(h http.Handler) http.Handler { return g.BusinessRead(h) },
				g.RequireEventTenant(deps.DataStore),
			)(listFencesHandler(deps.DataStore)).ServeHTTP(w, r)
		case eventRouteAttendanceList:
			Chain(
				g.AllowMethods(http.MethodGet),
				func(h http.Handler) http.Handler { return g.BusinessRead(h) },
				g.RequireEventTenant(deps.DataStore),
			)(listAttendanceHandler(deps.DataStore)).ServeHTTP(w, r)
		case eventRouteConsent:
			Chain(
				g.AllowMethods(http.MethodGet, http.MethodPut),
				func(h http.Handler) http.Handler { return g.BusinessRead(h) },
				g.RequireEventTenant(deps.DataStore),
			)(consentTemplateHandler(deps.DataStore)).ServeHTTP(w, r)
		case eventRouteShare:
			Chain(
				g.AllowMethods(http.MethodGet),
				func(h http.Handler) http.Handler { return g.BusinessRead(h) },
				g.RequireEventTenant(deps.DataStore),
			)(getEventShareHandler(deps.DataStore, deps.JoinPublicBase)).ServeHTTP(w, r)
		case eventRouteShareRegenerate:
			Chain(
				g.AllowMethods(http.MethodPost),
				func(h http.Handler) http.Handler { return g.BusinessWrite(h) },
				g.RequireEventTenant(deps.DataStore),
			)(regenerateEventQRHandler(deps.DataStore, deps.JoinPublicBase)).ServeHTTP(w, r)
		case eventRouteClockInShare:
			Chain(
				g.AllowMethods(http.MethodGet),
				func(h http.Handler) http.Handler { return g.BusinessRead(h) },
				g.RequireEventTenant(deps.DataStore),
			)(getEventClockInShareHandler(deps.DataStore, deps.JoinPublicBase)).ServeHTTP(w, r)
		case eventRouteClockInSettings:
			Chain(
				g.AllowMethods(http.MethodPatch),
				func(h http.Handler) http.Handler { return g.BusinessWrite(h) },
				g.RequireEventTenant(deps.DataStore),
			)(patchEventClockInSettingsHandler(deps.DataStore, deps.JoinPublicBase)).ServeHTTP(w, r)
		case eventRouteClockInShareRegenerate:
			Chain(
				g.AllowMethods(http.MethodPost),
				func(h http.Handler) http.Handler { return g.BusinessWrite(h) },
				g.RequireEventTenant(deps.DataStore),
			)(regenerateEventClockInShareHandler(deps.DataStore, deps.JoinPublicBase)).ServeHTTP(w, r)
		case eventRouteEventAnalytics:
			Chain(
				g.AllowMethods(http.MethodGet),
				func(h http.Handler) http.Handler { return g.BusinessRead(h) },
				g.RequireEventTenant(deps.DataStore),
			)(getEventAnalyticsHandler(deps.DataStore)).ServeHTTP(w, r)
		case eventRouteFenceDelete:
			Chain(
				g.AllowMethods(http.MethodDelete),
				func(h http.Handler) http.Handler { return g.BusinessWrite(h) },
				g.RequireEventTenant(deps.DataStore),
			)(deleteFenceHandler(deps.DataStore)).ServeHTTP(w, r)
		case eventRouteFenceAnalytics:
			Chain(
				g.AllowMethods(http.MethodGet),
				func(h http.Handler) http.Handler { return g.BusinessRead(h) },
				g.RequireEventTenant(deps.DataStore),
			)(getFenceAnalyticsHandler(deps.DataStore)).ServeHTTP(w, r)
		case eventRouteFenceCapture:
			Chain(
				g.AllowMethods(http.MethodGet, http.MethodPost),
				func(h http.Handler) http.Handler { return g.BusinessRead(h) },
				g.RequireEventTenant(deps.DataStore),
			)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodGet:
					getFenceCaptureHandler(deps.DataStore, deps.JoinPublicBase)(w, r)
				case http.MethodPost:
					createFenceCaptureHandler(deps.DataStore, deps.JoinPublicBase)(w, r)
				}
			})).ServeHTTP(w, r)
		case eventRouteFenceCaptureApply:
			Chain(
				g.AllowMethods(http.MethodPost),
				func(h http.Handler) http.Handler { return g.BusinessWrite(h) },
				g.RequireEventTenant(deps.DataStore),
			)(applyFenceCaptureHandler(deps.DataStore)).ServeHTTP(w, r)
		default:
			writeAPIError(w, http.StatusNotFound, "not_found", "Resource not found")
		}
	}))

	return mux
}

type eventRouteKind int

const (
	eventRouteUnknown eventRouteKind = iota
	eventRouteGet
	eventRouteFence
	eventRouteFencesList
	eventRouteFenceDelete
	eventRouteAttendanceList
	eventRouteConsent
	eventRouteShare
	eventRouteShareRegenerate
	eventRouteClockInShare
	eventRouteClockInSettings
	eventRouteClockInShareRegenerate
	eventRouteEventAnalytics
	eventRouteFenceAnalytics
	eventRouteFenceCapture
	eventRouteFenceCaptureApply
	eventRouteJoinQR
	eventRouteMarkPresent
)

func classifyEventRoute(r *http.Request) eventRouteKind {
	path := strings.TrimPrefix(r.URL.Path, "/v1/events/")
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")

	if len(parts) == 1 && r.Method == http.MethodGet {
		return eventRouteGet
	}
	if len(parts) == 2 && parts[1] == "fences" {
		if r.Method == http.MethodPost {
			return eventRouteFence
		}
		if r.Method == http.MethodGet {
			return eventRouteFencesList
		}
	}
	if len(parts) == 3 && parts[1] == "fences" && r.Method == http.MethodDelete {
		return eventRouteFenceDelete
	}
	if len(parts) == 2 && parts[1] == "attendance" && r.Method == http.MethodGet {
		return eventRouteAttendanceList
	}
	if len(parts) == 2 && parts[1] == "consent-template" {
		return eventRouteConsent
	}
	if len(parts) == 2 && parts[1] == "share" && r.Method == http.MethodGet {
		return eventRouteShare
	}
	if len(parts) == 3 && parts[1] == "share" && parts[2] == "regenerate" && r.Method == http.MethodPost {
		return eventRouteShareRegenerate
	}
	if len(parts) == 2 && parts[1] == "clock-in-share" && r.Method == http.MethodGet {
		return eventRouteClockInShare
	}
	if len(parts) == 2 && parts[1] == "clock-in-settings" && r.Method == http.MethodPatch {
		return eventRouteClockInSettings
	}
	if len(parts) == 3 && parts[1] == "clock-in-share" && parts[2] == "regenerate" && r.Method == http.MethodPost {
		return eventRouteClockInShareRegenerate
	}
	if len(parts) == 2 && parts[1] == "analytics" && r.Method == http.MethodGet {
		return eventRouteEventAnalytics
	}
	if len(parts) == 4 && parts[1] == "fences" && parts[3] == "analytics" && r.Method == http.MethodGet {
		return eventRouteFenceAnalytics
	}
	if len(parts) == 2 && parts[1] == "fence-capture" {
		if r.Method == http.MethodGet || r.Method == http.MethodPost {
			return eventRouteFenceCapture
		}
	}
	if len(parts) == 3 && parts[1] == "fence-capture" && parts[2] == "apply" && r.Method == http.MethodPost {
		return eventRouteFenceCaptureApply
	}
	if len(parts) == 2 && parts[1] == "join-by-qr" && r.Method == http.MethodPost {
		return eventRouteJoinQR
	}
	if len(parts) == 3 && parts[1] == "attendance" && parts[2] == "mark-present" && r.Method == http.MethodPost {
		return eventRouteMarkPresent
	}
	return eventRouteUnknown
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	writeJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
}

func dashboardEventsHandler(dataStore store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		biz, _ := businessAuthFromContext(r.Context())

		switch r.Method {
		case http.MethodPost:
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
				writeAPIError(w, http.StatusBadRequest, "invalid_datetime", "start_at must be a valid RFC3339 datetime")
				return
			}
			endAt, err := parseRFC3339Time(req.EndAt, "end_at")
			if err != nil {
				writeAPIError(w, http.StatusBadRequest, "invalid_datetime", "end_at must be a valid RFC3339 datetime")
				return
			}
			if err := domain.ValidateEventSchedule(startAt, endAt); err != nil {
				writeAPIError(w, http.StatusBadRequest, "invalid_schedule", err.Error())
				return
			}
			event, err := dataStore.CreateEvent(biz.OrganizationID, biz.UserID, req.Title, req.Description, startAt, endAt)
			if err != nil {
				writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
				return
			}
			writeJSON(w, http.StatusCreated, event)
		case http.MethodGet:
			events, err := dataStore.ListEventsByOrganization(biz.OrganizationID)
			if err != nil {
				writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
				return
			}
			writeJSON(w, http.StatusOK, ListEventsResponse{Events: events})
		}
	}
}

func dashboardEventHandler(dataStore store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		biz, _ := businessAuthFromContext(r.Context())
		eventID := eventIDFromPath(r.URL.Path)

		if classifyEventRoute(r) == eventRouteGet {
			event, found, err := dataStore.GetEventForOrganization(eventID, biz.OrganizationID)
			if err != nil {
				writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
				return
			}
			if !found {
				writeAPIError(w, http.StatusNotFound, "event_not_found", "Event was not found")
				return
			}
			writeJSON(w, http.StatusOK, event)
			return
		}

		var req CreateFenceRequest
		if err := decodeJSON(r, &req); err != nil {
			writeInvalidJSON(w, err)
			return
		}
		event, found, err := dataStore.GetEventForOrganization(eventID, biz.OrganizationID)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		if !found {
			writeAPIError(w, http.StatusNotFound, "event_not_found", "Event was not found")
			return
		}
		fenceInput, err := resolveCreateFenceInput(event, req)
		if err != nil {
			code := "invalid_fence"
			if strings.Contains(err.Error(), "start_at") || strings.Contains(err.Error(), "end_at") || strings.Contains(err.Error(), "schedule") {
				code = "invalid_schedule"
			}
			writeAPIError(w, http.StatusBadRequest, code, err.Error())
			return
		}
		fence, err := dataStore.AddFence(eventID, fenceInput)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, fence)
	}
}

func mobileEventHandler(dataStore store.Store, identityStore *mysqlstore.IdentityStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, _ := userAuthFromContext(r.Context())
		eventID := eventIDFromPath(r.URL.Path)
		path := strings.TrimPrefix(r.URL.Path, "/v1/events/")
		parts := strings.Split(strings.Trim(path, "/"), "/")

		if len(parts) == 2 && parts[1] == "join-by-qr" {
			var req JoinByQRRequest
			if err := decodeJSON(r, &req); err != nil {
				writeInvalidJSON(w, err)
				return
			}
			if strings.TrimSpace(req.QRToken) == "" {
				writeAPIError(w, http.StatusBadRequest, "qr_token_required", "qr_token is required")
				return
			}
			joinUserID := user.UserID
			if strings.TrimSpace(joinUserID) == "" {
				email := strings.TrimSpace(strings.ToLower(req.Email))
				first := strings.TrimSpace(req.FirstName)
				last := strings.TrimSpace(req.LastName)
				if email == "" || first == "" {
					writeAPIError(
						w,
						http.StatusBadRequest,
						"attendee_details_required",
						"email and first_name are required when not signed in",
					)
					return
				}
				if existing, ok, err := identityStore.GetUserByEmail(email); err == nil && ok {
					joinUserID = existing.ID
				} else if err != nil {
					writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
					return
				} else {
					created, err := identityStore.CreateUser(email, first, last, "")
					if err != nil {
						writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
						return
					}
					joinUserID = created.ID
				}
			}
			join, err := dataStore.JoinByQR(eventID, joinUserID, req.QRToken)
			if err != nil {
				writeStoreErr(w, err)
				return
			}
			writeJSON(w, http.StatusCreated, join)
			return
		}

		var req MarkPresentRequest
		if err := decodeJSON(r, &req); err != nil {
			writeInvalidJSON(w, err)
			return
		}
		if err := validateStrictLocationReport(req.Lat, req.Lng, req.AccuracyM, req.MockLocation); err != nil {
			writeLocationValidationErr(w, err)
			return
		}
		source := req.Source
		if strings.TrimSpace(source) == "" {
			source = "geofence_prompt"
		}
		record, err := dataStore.MarkPresent(eventID, user.UserID, source, req.Lat, req.Lng, req.AccuracyM)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, record)
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
