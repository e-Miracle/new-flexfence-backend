package store

import (
	"errors"
	"time"

	"github.com/flexfence/flexfence-backend/internal/domain"
)

var (
	ErrEventNotFound         = errors.New("event_not_found")
	ErrOTPChallengeNotFound  = errors.New("otp_challenge_not_found")
	ErrOTPInvalid            = errors.New("otp_invalid")
	ErrOTPExpired            = errors.New("otp_expired")
	ErrOTPConsumed           = errors.New("otp_consumed")
	ErrOTPTooManyAttempts    = errors.New("otp_too_many_attempts")
	ErrAlreadyMarked       = errors.New("already_marked_present")
	ErrInvalidQRToken      = errors.New("invalid_qr_token")
	ErrInvalidStorageState = errors.New("invalid_storage_state")
	ErrInvalidCredentials  = errors.New("invalid_credentials")
	ErrAccountDisabled     = errors.New("account_disabled")
	ErrInvalidSchedule     = errors.New("invalid_schedule")
	ErrAlreadyExists       = errors.New("already_exists")
	ErrJoinNotFound        = errors.New("join_not_found")
	ErrUserNotFound        = errors.New("user_not_found")
	ErrOpenSessionNotFound = errors.New("open_session_not_found")
	ErrCaptureNotFound     = errors.New("capture_session_not_found")
	ErrCaptureExpired      = errors.New("capture_session_expired")
	ErrInvalidCapture      = errors.New("invalid_capture")
	ErrEventLive           = errors.New("event_live")
	ErrFenceNotFound       = errors.New("fence_not_found")
)

type UserEventJoinFilter struct {
	Search     string
	Status     string
	JoinSource string
	StartFrom  *time.Time
	StartTo    *time.Time
	Page       int
	Limit      int
}

type UserEventJoinPage struct {
	Joins   []domain.UserEventJoin
	Total   int
	Page    int
	Limit   int
	HasMore bool
}

type ActivityHistoryFilter struct {
	Period string
	From   *time.Time
	To     *time.Time
}

type Store interface {
	CreateEvent(organizationID, createdByID, title, description string, startAt, endAt time.Time) (domain.Event, error)
	ListEvents() ([]domain.Event, error)
	ListEventsByOrganization(organizationID string) ([]domain.Event, error)
	GetEvent(eventID string) (domain.Event, bool, error)
	GetEventForOrganization(eventID, organizationID string) (domain.Event, bool, error)
	EnsureEventQRToken(eventID string) (string, error)
	RegenerateEventQRToken(eventID string) (string, error)
	AddFence(eventID string, in domain.FenceCreateInput) (domain.Fence, error)
	DeleteFence(eventID, fenceID string) error
	ListFencesByEvent(eventID string) ([]domain.Fence, error)
	GetFenceByEvent(eventID, fenceID string) (domain.Fence, bool, error)
	ProcessPendingEventGoLive(now time.Time) (int, error)
	ConsumeUserGeofenceAlerts(userID string) ([]domain.GeofenceAlert, error)
	ListJoinsByEvent(eventID string) ([]domain.EventJoin, error)
	ListJoinsByUser(userID string) ([]domain.UserEventJoin, error)
	ListJoinsByUserFiltered(userID string, filter UserEventJoinFilter) (UserEventJoinPage, error)
	ListSubscribedGeofenceEvents(userID string, now time.Time) ([]domain.SubscribedGeofenceEvent, error)
	DeleteUserEventJoin(userID, joinID string) error
	ListAttendanceByEvent(eventID string) ([]domain.AttendanceWithUser, error)
	GetEventAnalytics(eventID string) (domain.EventAnalytics, error)
	GetFenceAnalytics(eventID, fenceID string) (domain.FenceAnalytics, error)
	GetConsentTemplate(eventID string) (domain.ConsentTemplate, bool, error)
	SaveConsentTemplate(eventID string, tpl domain.ConsentTemplate) (domain.ConsentTemplate, error)
	RecordOrganizationConsentFields(organizationID string, fields []domain.ConsentField) error
	ListConsentFieldRecommendations(organizationID string, limit int) ([]domain.ConsentFieldRecommendation, error)
	JoinByQR(eventID, userID, qrToken string) (domain.EventJoin, error)
	MarkPresent(eventID, userID, source string, lat, lng, accuracyM float64) (domain.AttendanceRecord, error)
	ListUserActivityHistory(userID string, filter ActivityHistoryFilter) ([]domain.UserActivitySession, error)
	RecordClockIn(userID, eventID, eventTitle, fenceID, fenceName, source string, lat, lng float64) (domain.UserActivitySession, error)
	RecordClockOut(userID, sessionID, eventID, fenceID string) (domain.UserActivitySession, error)
	DeleteActivityHistoryOlderThan(cutoff time.Time) (int64, error)
	CreateFenceCaptureSession(eventID, targetShape string) (domain.FenceCaptureSession, error)
	GetActiveFenceCaptureSession(eventID string) (domain.FenceCaptureSession, bool, error)
	GetFenceCaptureSessionByToken(token string) (domain.FenceCaptureSession, bool, error)
	AppendFenceCapturePoint(token string, point domain.FenceCapturePoint) (domain.FenceCaptureSession, error)
	ApplyFenceCaptureSession(eventID, sessionID, name string, startAt, endAt time.Time) (domain.Fence, error)
}
