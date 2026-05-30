package http

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/flexfence/flexfence-backend/internal/auth"
	"github.com/flexfence/flexfence-backend/internal/domain"
	"github.com/flexfence/flexfence-backend/internal/mail"
	"github.com/flexfence/flexfence-backend/internal/store"
	mysqlstore "github.com/flexfence/flexfence-backend/internal/store/mysql"
)

type RouterDeps struct {
	DataStore        store.Store
	IdentityStore    *mysqlstore.IdentityStore
	Tokens           *auth.TokenService
	Mailer           mail.Mailer
	GoogleClient     string
	OTPLength        int
	OTPExpireMinutes int
	DashboardURL     string
	JoinPublicBase   string
}

func businessLoginHandler(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		var req BusinessLoginRequest
		if err := decodeJSON(r, &req); err != nil {
			writeInvalidJSON(w, err)
			return
		}
		if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Password) == "" {
			writeAPIError(w, http.StatusBadRequest, "email_and_password_required", "email and password are required")
			return
		}

		record, ok, err := deps.IdentityStore.GetBusinessAuthByEmail(strings.TrimSpace(req.Email))
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		if !ok || !auth.CheckPassword(record.PasswordHash, req.Password) {
			writeAPIError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
			return
		}
		if record.User.Status != domain.StatusActive {
			writeAPIError(w, http.StatusForbidden, "account_disabled", "Account is disabled")
			return
		}

		resp, err := issueAndSendBusinessOTP(r.Context(), deps, record.User.ID, "")
		if err != nil {
			if errors.Is(err, errEmailSendFailed) {
				writeAPIError(w, http.StatusBadGateway, "email_send_failed", "Could not send verification email; check SMTP settings")
				return
			}
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func businessRegisterHandler(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		var req BusinessRegisterRequest
		if err := decodeJSON(r, &req); err != nil {
			writeInvalidJSON(w, err)
			return
		}

		req.OrganizationName = strings.TrimSpace(req.OrganizationName)
		req.FirstName = strings.TrimSpace(req.FirstName)
		req.LastName = strings.TrimSpace(req.LastName)
		req.Email = strings.TrimSpace(strings.ToLower(req.Email))
		if req.OrganizationName == "" || req.FirstName == "" || req.Email == "" || req.Password == "" {
			writeAPIError(
				w,
				http.StatusBadRequest,
				"registration_fields_required",
				"organization_name, first_name, email, and password are required",
			)
			return
		}
		if pwdErr := auth.ValidatePassword(req.Password); pwdErr != nil {
			writeAPIError(w, http.StatusBadRequest, pwdErr.Code, pwdErr.Message)
			return
		}
		if !strings.Contains(req.Email, "@") {
			writeAPIError(w, http.StatusBadRequest, "invalid_email", "email must be valid")
			return
		}

		passwordHash, err := auth.HashPassword(req.Password)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		user, err := deps.IdentityStore.RegisterBusinessOwner(
			req.OrganizationName,
			req.Email,
			passwordHash,
			req.FirstName,
			req.LastName,
		)
		if err != nil {
			if errors.Is(err, store.ErrAlreadyExists) {
				writeAPIError(w, http.StatusConflict, "email_already_registered", "A business account with this email already exists")
				return
			}
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}

		resp, err := issueAndSendBusinessOTP(r.Context(), deps, user.ID, "")
		if err != nil {
			if errors.Is(err, errEmailSendFailed) {
				writeAPIError(w, http.StatusBadGateway, "email_send_failed", "Could not send verification email; check SMTP settings")
				return
			}
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		writeJSON(w, http.StatusCreated, resp)
	}
}

func issueUserLoginResponse(w http.ResponseWriter, deps RouterDeps, user domain.User) bool {
	token, expiresAt, err := deps.Tokens.IssueUserToken(user.ID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
		return false
	}
	writeJSON(w, http.StatusOK, UserLoginResponse{
		AuthTokenResponse: AuthTokenResponse{
			AccessToken: token,
			TokenType:   "Bearer",
			ExpiresAt:   expiresAt.Format(time.RFC3339),
		},
		User: UserProfile{
			ID:        user.ID,
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
		},
	})
	return true
}

func userRegisterHandler(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		var req UserRegisterRequest
		if err := decodeJSON(r, &req); err != nil {
			writeInvalidJSON(w, err)
			return
		}
		req.Email = strings.TrimSpace(strings.ToLower(req.Email))
		req.FirstName = strings.TrimSpace(req.FirstName)
		req.LastName = strings.TrimSpace(req.LastName)
		if req.Email == "" || req.Password == "" || req.FirstName == "" {
			writeAPIError(
				w,
				http.StatusBadRequest,
				"registration_fields_required",
				"email, password, and first_name are required",
			)
			return
		}
		if pwdErr := auth.ValidatePassword(req.Password); pwdErr != nil {
			writeAPIError(w, http.StatusBadRequest, pwdErr.Code, pwdErr.Message)
			return
		}
		if !strings.Contains(req.Email, "@") {
			writeAPIError(w, http.StatusBadRequest, "invalid_email", "email must be valid")
			return
		}

		passwordHash, err := auth.HashPassword(req.Password)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		user, err := deps.IdentityStore.RegisterUserWithPassword(
			req.Email,
			passwordHash,
			req.FirstName,
			req.LastName,
			strings.TrimSpace(req.Phone),
		)
		if err != nil {
			if errors.Is(err, store.ErrAlreadyExists) {
				writeAPIError(w, http.StatusConflict, "email_already_registered", "An account with this email already exists")
				return
			}
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		token, expiresAt, err := deps.Tokens.IssueUserToken(user.ID)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		writeJSON(w, http.StatusCreated, UserLoginResponse{
			AuthTokenResponse: AuthTokenResponse{
				AccessToken: token,
				TokenType:   "Bearer",
				ExpiresAt:   expiresAt.Format(time.RFC3339),
			},
			User: UserProfile{
				ID:        user.ID,
				Email:     user.Email,
				FirstName: user.FirstName,
				LastName:  user.LastName,
			},
		})
	}
}

func userLoginHandler(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		var req UserLoginRequest
		if err := decodeJSON(r, &req); err != nil {
			writeInvalidJSON(w, err)
			return
		}
		if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Password) == "" {
			writeAPIError(w, http.StatusBadRequest, "email_and_password_required", "email and password are required")
			return
		}

		record, ok, err := deps.IdentityStore.GetUserAuthByEmail(strings.TrimSpace(req.Email))
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		if !ok || record.PasswordHash == "" || !auth.CheckPassword(record.PasswordHash, req.Password) {
			writeAPIError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
			return
		}
		if record.User.Status != domain.StatusActive {
			writeAPIError(w, http.StatusForbidden, "account_disabled", "Account is disabled")
			return
		}
		issueUserLoginResponse(w, deps, record.User)
	}
}

func userGoogleOAuthHandler(deps RouterDeps) http.HandlerFunc {
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

		user, err := deps.IdentityStore.UpsertUserFromGoogle(profile.Sub, profile.Email, profile.FirstName, profile.LastName)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		if user.Status != domain.StatusActive {
			writeAPIError(w, http.StatusForbidden, "account_disabled", "Account is disabled")
			return
		}

		issueUserLoginResponse(w, deps, user)
	}
}

func verifyGoogleRequest(deps RouterDeps, req GoogleOAuthRequest) (auth.GoogleProfile, error) {
	profile, err := auth.VerifyGoogleIDToken(
		deps.GoogleClient,
		req.IDToken,
		req.GoogleSub,
		req.Email,
		req.FirstName,
		req.LastName,
	)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidGoogleToken) {
			return auth.GoogleProfile{}, err
		}
		return auth.GoogleProfile{}, err
	}
	return profile, nil
}

func mapBusinessProfile(u domain.BusinessUser) BusinessUserProfile {
	return BusinessUserProfile{
		ID:             u.ID,
		OrganizationID: u.OrganizationID,
		Email:          u.Email,
		FirstName:      u.FirstName,
		LastName:       u.LastName,
		Role:           u.Role,
	}
}
