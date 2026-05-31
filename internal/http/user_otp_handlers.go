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
)

const userRegistrationOTPLength = 6

type UserOTPChallengeResponse struct {
	OTPRequired bool   `json:"otp_required"`
	ChallengeID string `json:"challenge_id"`
	MaskedEmail string `json:"masked_email"`
	ExpiresAt   string `json:"expires_at"`
}

type UserOTPVerifyRequest struct {
	ChallengeID string `json:"challenge_id"`
	Code        string `json:"code"`
}

type UserOTPResendRequest struct {
	ChallengeID string `json:"challenge_id"`
}

func userOTPVerifyHandler(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		var req UserOTPVerifyRequest
		if err := decodeJSON(r, &req); err != nil {
			writeInvalidJSON(w, err)
			return
		}
		if strings.TrimSpace(req.ChallengeID) == "" || strings.TrimSpace(req.Code) == "" {
			writeAPIError(w, http.StatusBadRequest, "challenge_and_code_required", "challenge_id and code are required")
			return
		}

		user, err := deps.IdentityStore.VerifyUserOTPChallenge(strings.TrimSpace(req.ChallengeID), strings.TrimSpace(req.Code))
		if err != nil {
			writeOTPErr(w, err)
			return
		}
		if user.Status != domain.StatusActive {
			writeAPIError(w, http.StatusForbidden, "account_disabled", "Account is disabled")
			return
		}
		issueUserLoginResponse(w, deps, user)
	}
}

func userOTPResendHandler(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		var req UserOTPResendRequest
		if err := decodeJSON(r, &req); err != nil {
			writeInvalidJSON(w, err)
			return
		}
		if strings.TrimSpace(req.ChallengeID) == "" {
			writeAPIError(w, http.StatusBadRequest, "challenge_id_required", "challenge_id is required")
			return
		}

		resp, err := issueAndSendUserOTP(r.Context(), deps, "", strings.TrimSpace(req.ChallengeID))
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

func issueAndSendUserOTP(ctx context.Context, deps RouterDeps, userID, existingChallengeID string) (UserOTPChallengeResponse, error) {
	code, err := auth.GenerateNumericOTP(userRegistrationOTPLength)
	if err != nil {
		return UserOTPChallengeResponse{}, err
	}
	ttl := time.Duration(deps.OTPExpireMinutes) * time.Minute

	var (
		challengeID string
		expiresAt   time.Time
		email       string
	)
	if existingChallengeID != "" {
		challengeID = existingChallengeID
		expiresAt, userID, email, err = deps.IdentityStore.ResendUserOTPChallenge(existingChallengeID, code, ttl)
	} else {
		challengeID, expiresAt, err = deps.IdentityStore.CreateUserOTPChallenge(userID, code, ttl)
		if err != nil {
			return UserOTPChallengeResponse{}, err
		}
		email, err = deps.IdentityStore.GetUserOTPChallengeEmail(challengeID)
	}
	if err != nil {
		return UserOTPChallengeResponse{}, err
	}

	if err := mail.SendUserEmailVerificationOTP(ctx, deps.Mailer, email, code, deps.OTPExpireMinutes); err != nil {
		return UserOTPChallengeResponse{}, errEmailSendFailed
	}

	return UserOTPChallengeResponse{
		OTPRequired: true,
		ChallengeID: challengeID,
		MaskedEmail: auth.MaskEmail(email),
		ExpiresAt:   expiresAt.Format(time.RFC3339),
	}, nil
}
