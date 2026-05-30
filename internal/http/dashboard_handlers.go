package http

import (
	"errors"
	"net/http"
	"time"

	"github.com/flexfence/flexfence-backend/internal/auth"
	"github.com/flexfence/flexfence-backend/internal/domain"
	"github.com/flexfence/flexfence-backend/internal/store"
)

func listFencesHandler(dataStore store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventID := eventIDFromPath(r.URL.Path)
		fences, err := dataStore.ListFencesByEvent(eventID)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"fences": fences})
	}
}

func listAttendanceHandler(dataStore store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventID := eventIDFromPath(r.URL.Path)
		records, err := dataStore.ListAttendanceByEvent(eventID)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"attendance": records})
	}
}

func consentTemplateHandler(dataStore store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventID := eventIDFromPath(r.URL.Path)
		switch r.Method {
		case http.MethodGet:
			tpl, ok, err := dataStore.GetConsentTemplate(eventID)
			if err != nil {
				writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
				return
			}
			if !ok {
				writeJSON(w, http.StatusOK, defaultConsentTemplate(eventID))
				return
			}
			writeJSON(w, http.StatusOK, tpl)
		case http.MethodPut:
			biz, ok := businessAuthFromContext(r.Context())
			if !ok {
				writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
				return
			}
			var req ConsentTemplateRequest
			if err := decodeJSON(r, &req); err != nil {
				writeInvalidJSON(w, err)
				return
			}
			normalized, err := domain.NormalizeConsentFields(req.RequiredFields)
			if err != nil {
				writeAPIError(w, http.StatusBadRequest, "invalid_consent_fields", err.Error())
				return
			}
			saved, err := dataStore.SaveConsentTemplate(eventID, domain.ConsentTemplate{
				RequiredFields: normalized,
				TrackEntryExit: req.TrackEntryExit,
				TrackMovement:  req.TrackMovement,
			})
			if err != nil {
				writeStoreErr(w, err)
				return
			}
			if err := dataStore.RecordOrganizationConsentFields(biz.OrganizationID, normalized); err != nil {
				writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
				return
			}
			writeJSON(w, http.StatusOK, saved)
		default:
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		}
	}
}

func defaultConsentTemplate(eventID string) domain.ConsentTemplate {
	return domain.ConsentTemplate{
		EventID: eventID,
		RequiredFields: []domain.ConsentField{
			{Key: "email", Label: "Email", Required: true, ValueType: domain.ConsentValueEmail},
			{Key: "first_name", Label: "First name", Required: true, ValueType: domain.ConsentValueText},
			{Key: "last_name", Label: "Last name", Required: false, ValueType: domain.ConsentValueText},
		},
		TrackEntryExit: true,
		TrackMovement:  false,
	}
}

func listConsentFieldRecommendationsHandler(dataStore store.Store) http.HandlerFunc {
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
		recs, err := dataStore.ListConsentFieldRecommendations(biz.OrganizationID, 10)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		if recs == nil {
			recs = []domain.ConsentFieldRecommendation{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"recommendations": recs})
	}
}

func businessGoogleOAuthHandler(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		var req GoogleOAuthRequest
		if err := decodeJSON(r, &req); err != nil {
			writeInvalidJSON(w, err)
			return
		}

		profile, err := verifyGoogleRequest(deps, req)
		if err != nil {
			if errors.Is(err, auth.ErrInvalidGoogleToken) {
				writeAPIError(w, http.StatusBadRequest, "invalid_google_token", auth.DevModeHint())
				return
			}
			writeAPIError(w, http.StatusBadGateway, "google_verification_failed", "Could not verify Google token")
			return
		}

		user, err := deps.IdentityStore.LoginBusinessWithGoogle(profile.Sub, profile.Email, profile.FirstName, profile.LastName)
		if err != nil {
			if errors.Is(err, store.ErrInvalidCredentials) {
				writeAPIError(w, http.StatusUnauthorized, "invalid_credentials", "No business account found for this Google email")
				return
			}
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		if user.Status != domain.StatusActive {
			writeAPIError(w, http.StatusForbidden, "account_disabled", "Account is disabled")
			return
		}

		token, expiresAt, err := deps.Tokens.IssueBusinessToken(user.ID, user.OrganizationID, user.Role)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		_ = deps.IdentityStore.TouchBusinessLogin(user.ID)

		writeJSON(w, http.StatusOK, BusinessLoginResponse{
			AuthTokenResponse: AuthTokenResponse{
				AccessToken: token,
				TokenType:   "Bearer",
				ExpiresAt:   expiresAt.Format(time.RFC3339),
			},
			User: mapBusinessProfile(user),
		})
	}
}
