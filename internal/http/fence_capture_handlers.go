package http

import (
	"net/http"
	"strings"
	"time"

	"github.com/flexfence/flexfence-backend/internal/domain"
	"github.com/flexfence/flexfence-backend/internal/store"
)

type FenceCaptureLinkResponse struct {
	Session       domain.FenceCaptureSession `json:"session"`
	CaptureLink   string                     `json:"capture_link"`
	DeepLink      string                     `json:"deep_link"`
	QRCodePayload string                     `json:"qr_code_payload"`
}

type SubmitCapturePointRequest struct {
	Lat          float64 `json:"lat"`
	Lng          float64 `json:"lng"`
	AccuracyM    float64 `json:"accuracy_m"`
	Role         string  `json:"role,omitempty"`
	MockLocation bool    `json:"mock_location,omitempty"`
}

type CreateFenceCaptureRequest struct {
	TargetShape string `json:"target_shape"`
}

type ApplyFenceCaptureRequest struct {
	SessionID string `json:"session_id"`
	Name      string `json:"name"`
	StartAt   string `json:"start_at,omitempty"`
	EndAt     string `json:"end_at,omitempty"`
}

func createFenceCaptureHandler(dataStore store.Store, joinPublicBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		eventID := strings.TrimPrefix(r.URL.Path, "/v1/events/")
		eventID = strings.TrimSuffix(eventID, "/fence-capture")
		eventID = strings.Trim(eventID, "/")
		var req CreateFenceCaptureRequest
		_ = decodeJSON(r, &req)
		session, err := dataStore.CreateFenceCaptureSession(eventID, req.TargetShape)
		if err != nil {
			if err == store.ErrEventNotFound {
				writeAPIError(w, http.StatusNotFound, "event_not_found", "Event not found")
				return
			}
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		writeJSON(w, http.StatusCreated, FenceCaptureLinkResponse{
			Session:       session,
			CaptureLink:   buildFenceCaptureLink(eventID, session.Token, joinPublicBase),
			DeepLink:      buildFenceCaptureDeepLink(eventID, session.Token),
			QRCodePayload: buildFenceCaptureDeepLink(eventID, session.Token),
		})
	}
}

func getFenceCaptureHandler(dataStore store.Store, joinPublicBase string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		eventID := strings.TrimPrefix(r.URL.Path, "/v1/events/")
		eventID = strings.TrimSuffix(eventID, "/fence-capture")
		eventID = strings.Trim(eventID, "/")
		session, ok, err := dataStore.GetActiveFenceCaptureSession(eventID)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		if !ok {
			writeJSON(w, http.StatusOK, FenceCaptureLinkResponse{
				Session: domain.FenceCaptureSession{EventID: eventID, Points: []domain.FenceCapturePoint{}},
			})
			return
		}
		writeJSON(w, http.StatusOK, FenceCaptureLinkResponse{
			Session:       session,
			CaptureLink:   buildFenceCaptureLink(eventID, session.Token, joinPublicBase),
			DeepLink:      buildFenceCaptureDeepLink(eventID, session.Token),
			QRCodePayload: buildFenceCaptureDeepLink(eventID, session.Token),
		})
	}
}

func applyFenceCaptureHandler(dataStore store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		eventID := strings.TrimPrefix(r.URL.Path, "/v1/events/")
		eventID = strings.TrimSuffix(eventID, "/fence-capture/apply")
		eventID = strings.Trim(eventID, "/")

		var req ApplyFenceCaptureRequest
		if err := decodeJSON(r, &req); err != nil {
			writeInvalidJSON(w, err)
			return
		}
		if strings.TrimSpace(req.SessionID) == "" {
			writeAPIError(w, http.StatusBadRequest, "session_id_required", "session_id is required")
			return
		}

		var startAt, endAt time.Time
		if raw := strings.TrimSpace(req.StartAt); raw != "" {
			t, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				writeAPIError(w, http.StatusBadRequest, "invalid_datetime", "start_at must be RFC3339")
				return
			}
			startAt = t
		}
		if raw := strings.TrimSpace(req.EndAt); raw != "" {
			t, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				writeAPIError(w, http.StatusBadRequest, "invalid_datetime", "end_at must be RFC3339")
				return
			}
			endAt = t
		}

		fence, err := dataStore.ApplyFenceCaptureSession(eventID, req.SessionID, req.Name, startAt, endAt)
		if err != nil {
			switch err {
			case store.ErrCaptureNotFound:
				writeAPIError(w, http.StatusNotFound, "capture_not_found", "Capture session not found")
			case store.ErrEventNotFound:
				writeAPIError(w, http.StatusNotFound, "event_not_found", "Event not found")
			case store.ErrInvalidSchedule, store.ErrInvalidCapture:
				writeAPIError(w, http.StatusBadRequest, "invalid_capture", "Capture points are insufficient for this fence shape")
			default:
				writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			}
			return
		}
		writeJSON(w, http.StatusCreated, fence)
	}
}

func publicGetFenceCaptureHandler(dataStore store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		token := strings.TrimPrefix(r.URL.Path, "/v1/public/fence-capture/")
		token = strings.Trim(token, "/")
		session, ok, err := dataStore.GetFenceCaptureSessionByToken(token)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		if !ok {
			writeAPIError(w, http.StatusNotFound, "capture_not_found", "Capture link is invalid or expired")
			return
		}
		writeJSON(w, http.StatusOK, session)
	}
}

func publicSubmitCapturePointHandler(dataStore store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		token := strings.TrimPrefix(r.URL.Path, "/v1/public/fence-capture/")
		token = strings.TrimSuffix(token, "/points")
		token = strings.Trim(token, "/")

		var req SubmitCapturePointRequest
		if err := decodeJSON(r, &req); err != nil {
			writeInvalidJSON(w, err)
			return
		}
		if req.Lat == 0 && req.Lng == 0 {
			writeAPIError(w, http.StatusBadRequest, "coordinates_required", "lat and lng are required")
			return
		}
		if err := validateStrictLocationReport(req.Lat, req.Lng, req.AccuracyM, req.MockLocation); err != nil {
			writeLocationValidationErr(w, err)
			return
		}

		session, err := dataStore.AppendFenceCapturePoint(token, domain.FenceCapturePoint{
			Lat:        req.Lat,
			Lng:        req.Lng,
			AccuracyM:  req.AccuracyM,
			Role:       req.Role,
			CapturedAt: time.Now().UTC(),
		})
		if err != nil {
			switch err {
			case store.ErrCaptureNotFound:
				writeAPIError(w, http.StatusNotFound, "capture_not_found", "Capture link is invalid or expired")
			case store.ErrCaptureExpired:
				writeAPIError(w, http.StatusGone, "capture_expired", "Capture link has expired")
			default:
				writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			}
			return
		}
		writeJSON(w, http.StatusOK, session)
	}
}
