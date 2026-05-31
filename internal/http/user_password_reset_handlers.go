package http

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/flexfence/flexfence-backend/internal/auth"
	"github.com/flexfence/flexfence-backend/internal/mail"
)

const userPasswordResetOTPLength = 6

type UserPasswordResetRequest struct {
	Email string `json:"email"`
}

type UserPasswordResetRequestResponse struct {
	Message     string `json:"message"`
	ChallengeID string `json:"challenge_id,omitempty"`
	MaskedEmail string `json:"masked_email,omitempty"`
	ExpiresAt   string `json:"expires_at,omitempty"`
}

type UserPasswordResetConfirmRequest struct {
	ChallengeID string `json:"challenge_id"`
	Code        string `json:"code"`
	NewPassword string `json:"new_password"`
}

type UserPasswordResetResendRequest struct {
	ChallengeID string `json:"challenge_id"`
}

func userPasswordResetRequestHandler(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		var req UserPasswordResetRequest
		if err := decodeJSON(r, &req); err != nil {
			writeInvalidJSON(w, err)
			return
		}
		email := strings.TrimSpace(strings.ToLower(req.Email))
		if email == "" || !strings.Contains(email, "@") {
			writeAPIError(w, http.StatusBadRequest, "invalid_email", "A valid email is required")
			return
		}

		record, ok, err := deps.IdentityStore.GetUserAuthByEmail(email)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		if !ok {
			writeJSON(w, http.StatusOK, UserPasswordResetRequestResponse{
				Message: "If an account exists for this email, a reset code has been sent.",
			})
			return
		}
		if record.PasswordHash == "" {
			if strings.TrimSpace(record.GoogleSub) != "" {
				writeAPIError(
					w,
					http.StatusBadRequest,
					"social_login_only",
					"This account uses Google sign-in. Sign in with Google instead of resetting a password.",
				)
				return
			}
			writeJSON(w, http.StatusOK, UserPasswordResetRequestResponse{
				Message: "If an account exists for this email, a reset code has been sent.",
			})
			return
		}

		resp, err := issueAndSendUserPasswordResetOTP(r.Context(), deps, record.User.ID, "")
		if err != nil {
			if errors.Is(err, errEmailSendFailed) {
				writeAPIError(w, http.StatusBadGateway, "email_send_failed", "Could not send reset email; check SMTP settings")
				return
			}
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		writeJSON(w, http.StatusOK, UserPasswordResetRequestResponse{
			Message:     "If an account exists for this email, a reset code has been sent.",
			ChallengeID: resp.ChallengeID,
			MaskedEmail: resp.MaskedEmail,
			ExpiresAt:   resp.ExpiresAt,
		})
	}
}

func userPasswordResetResendHandler(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		var req UserPasswordResetResendRequest
		if err := decodeJSON(r, &req); err != nil {
			writeInvalidJSON(w, err)
			return
		}
		if strings.TrimSpace(req.ChallengeID) == "" {
			writeAPIError(w, http.StatusBadRequest, "challenge_id_required", "challenge_id is required")
			return
		}

		resp, err := issueAndSendUserPasswordResetOTP(r.Context(), deps, "", strings.TrimSpace(req.ChallengeID))
		if err != nil {
			if errors.Is(err, errEmailSendFailed) {
				writeAPIError(w, http.StatusBadGateway, "email_send_failed", "Could not send reset email")
				return
			}
			writeOTPErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func userPasswordResetConfirmHandler(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		var req UserPasswordResetConfirmRequest
		if err := decodeJSON(r, &req); err != nil {
			writeInvalidJSON(w, err)
			return
		}
		challengeID := strings.TrimSpace(req.ChallengeID)
		code := strings.TrimSpace(req.Code)
		newPassword := strings.TrimSpace(req.NewPassword)
		if challengeID == "" || code == "" || newPassword == "" {
			writeAPIError(
				w,
				http.StatusBadRequest,
				"reset_fields_required",
				"challenge_id, code, and new_password are required",
			)
			return
		}
		if pwdErr := auth.ValidatePassword(newPassword); pwdErr != nil {
			writeAPIError(w, http.StatusBadRequest, pwdErr.Code, pwdErr.Message)
			return
		}

		hash, err := auth.HashPassword(newPassword)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		if err := deps.IdentityStore.ConfirmUserPasswordReset(challengeID, code, hash); err != nil {
			writeOTPErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func issueAndSendUserPasswordResetOTP(ctx context.Context, deps RouterDeps, userID, existingChallengeID string) (UserOTPChallengeResponse, error) {
	code, err := auth.GenerateNumericOTP(userPasswordResetOTPLength)
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
		expiresAt, userID, email, err = deps.IdentityStore.ResendUserPasswordResetChallenge(existingChallengeID, code, ttl)
	} else {
		challengeID, expiresAt, err = deps.IdentityStore.CreateUserPasswordResetChallenge(userID, code, ttl)
		if err != nil {
			return UserOTPChallengeResponse{}, err
		}
		email, err = deps.IdentityStore.GetUserPasswordResetChallengeEmail(challengeID)
	}
	if err != nil {
		return UserOTPChallengeResponse{}, err
	}

	if err := mail.SendUserPasswordResetOTP(ctx, deps.Mailer, email, code, deps.OTPExpireMinutes); err != nil {
		return UserOTPChallengeResponse{}, errEmailSendFailed
	}

	return UserOTPChallengeResponse{
		OTPRequired: true,
		ChallengeID: challengeID,
		MaskedEmail: auth.MaskEmail(email),
		ExpiresAt:   expiresAt.Format(time.RFC3339),
	}, nil
}
