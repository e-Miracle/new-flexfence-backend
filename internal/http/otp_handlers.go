package http

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/flexfence/flexfence-backend/internal/auth"
	"github.com/flexfence/flexfence-backend/internal/domain"
	"github.com/flexfence/flexfence-backend/internal/mail"
	"github.com/flexfence/flexfence-backend/internal/store"
)

func businessOTPVerifyHandler(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		var req BusinessOTPVerifyRequest
		if err := decodeJSON(r, &req); err != nil {
			writeInvalidJSON(w, err)
			return
		}
		if strings.TrimSpace(req.ChallengeID) == "" || strings.TrimSpace(req.Code) == "" {
			writeAPIError(w, http.StatusBadRequest, "challenge_and_code_required", "challenge_id and code are required")
			return
		}

		user, err := deps.IdentityStore.VerifyBusinessOTPChallenge(strings.TrimSpace(req.ChallengeID), strings.TrimSpace(req.Code))
		if err != nil {
			writeOTPErr(w, err)
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

func businessOTPResendHandler(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		var req BusinessOTPResendRequest
		if err := decodeJSON(r, &req); err != nil {
			writeInvalidJSON(w, err)
			return
		}
		if strings.TrimSpace(req.ChallengeID) == "" {
			writeAPIError(w, http.StatusBadRequest, "challenge_id_required", "challenge_id is required")
			return
		}

		resp, err := issueAndSendBusinessOTP(r.Context(), deps, "", strings.TrimSpace(req.ChallengeID))
		if err != nil {
			if errors.Is(err, errEmailSendFailed) {
				writeAPIError(w, http.StatusBadGateway, "email_send_failed", "Could not send verification email")
				return
			}
			writeOTPErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

var errEmailSendFailed = errors.New("email send failed")

func issueAndSendBusinessOTP(ctx context.Context, deps RouterDeps, businessUserID, existingChallengeID string) (BusinessLoginOTPChallengeResponse, error) {
	code, err := auth.GenerateNumericOTP(deps.OTPLength)
	if err != nil {
		return BusinessLoginOTPChallengeResponse{}, err
	}
	ttl := time.Duration(deps.OTPExpireMinutes) * time.Minute

	var (
		challengeID string
		expiresAt   time.Time
		email       string
	)
	if existingChallengeID != "" {
		challengeID = existingChallengeID
		expiresAt, businessUserID, email, err = deps.IdentityStore.ResendBusinessOTPChallenge(existingChallengeID, code, ttl)
	} else {
		challengeID, expiresAt, err = deps.IdentityStore.CreateBusinessOTPChallenge(businessUserID, code, ttl)
		if err != nil {
			return BusinessLoginOTPChallengeResponse{}, err
		}
		email, err = deps.IdentityStore.GetBusinessOTPChallengeEmail(challengeID)
	}
	if err != nil {
		return BusinessLoginOTPChallengeResponse{}, err
	}

	if err := mail.SendBusinessLoginOTP(ctx, deps.Mailer, email, code, deps.OTPExpireMinutes, deps.DashboardURL); err != nil {
		return BusinessLoginOTPChallengeResponse{}, errEmailSendFailed
	}

	return BusinessLoginOTPChallengeResponse{
		OTPRequired: true,
		ChallengeID: challengeID,
		MaskedEmail: auth.MaskEmail(email),
		ExpiresAt:   expiresAt.Format(time.RFC3339),
	}, nil
}

func writeOTPErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrOTPChallengeNotFound):
		writeAPIError(w, http.StatusNotFound, "otp_challenge_not_found", "Verification session not found")
	case errors.Is(err, store.ErrOTPInvalid):
		writeAPIError(w, http.StatusUnauthorized, "invalid_otp", "Invalid verification code")
	case errors.Is(err, store.ErrOTPExpired):
		writeAPIError(w, http.StatusUnauthorized, "otp_expired", "Verification code has expired")
	case errors.Is(err, store.ErrOTPConsumed):
		writeAPIError(w, http.StatusConflict, "otp_consumed", "Verification code was already used")
	case errors.Is(err, store.ErrOTPTooManyAttempts):
		writeAPIError(w, http.StatusTooManyRequests, "otp_too_many_attempts", "Too many failed attempts; request a new code")
	default:
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
	}
}
