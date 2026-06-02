package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/flexfence/flexfence-backend/internal/store"
)

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, APIError{
		Code:    code,
		Message: message,
	})
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func writeInvalidJSON(w http.ResponseWriter, err error) {
	writeAPIError(w, http.StatusBadRequest, "invalid_json", jsonDecodeMessage(err))
}

func jsonDecodeMessage(err error) string {
	if err == nil {
		return "Request body must be valid JSON"
	}
	if errors.Is(err, io.EOF) {
		return "Request body is empty; send JSON with Content-Type: application/json"
	}
	msg := err.Error()
	if strings.Contains(msg, "unknown field") {
		return msg + " (use snake_case field names, e.g. start_at not startAt)"
	}
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return "Request body is not valid JSON"
	}
	return "Request body must be valid JSON: " + msg
}

func writeStoreErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrEventNotFound):
		writeAPIError(w, http.StatusNotFound, "event_not_found", "Event was not found")
	case errors.Is(err, store.ErrAlreadyMarked):
		writeAPIError(w, http.StatusConflict, "already_marked_present", "User is already marked present for this event")
	case errors.Is(err, store.ErrInvalidQRToken):
		writeAPIError(w, http.StatusBadRequest, "invalid_qr_token", "QR token is invalid")
	case errors.Is(err, store.ErrClockInQRDisabled):
		writeAPIError(w, http.StatusBadRequest, "clock_in_qr_disabled", "Scan to clock-in is not enabled for this event")
	case errors.Is(err, store.ErrClockInQRExpired):
		writeAPIError(w, http.StatusBadRequest, "clock_in_qr_expired", "Clock-in QR code has expired")
	case errors.Is(err, store.ErrNotJoinedEvent):
		writeAPIError(w, http.StatusForbidden, "not_joined_event", "You must join the event before clocking in")
	case errors.Is(err, store.ErrConsentRequired):
		writeAPIError(w, http.StatusForbidden, "consent_required", "You must complete the event consent form before clocking in")
	case errors.Is(err, store.ErrQRScanRequired):
		writeAPIError(w, http.StatusBadRequest, "qr_scan_required", "Scan the event QR code to clock in")
	case errors.Is(err, store.ErrInvalidConsent):
		msg := strings.TrimPrefix(err.Error(), store.ErrInvalidConsent.Error()+": ")
		if msg == err.Error() {
			msg = "Consent submission is invalid"
		}
		writeAPIError(w, http.StatusBadRequest, "invalid_consent", msg)
	case errors.Is(err, store.ErrInvalidSchedule):
		writeAPIError(w, http.StatusBadRequest, "invalid_schedule", err.Error())
	case errors.Is(err, store.ErrEventLive):
		writeAPIError(w, http.StatusConflict, "event_live", "This action is not allowed while the event is live")
	case errors.Is(err, store.ErrFenceNotFound):
		writeAPIError(w, http.StatusNotFound, "fence_not_found", "Fence was not found")
	default:
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
	}
}

