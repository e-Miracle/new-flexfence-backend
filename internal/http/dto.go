package http

import "github.com/flexfence/flexfence-backend/internal/domain"

type HealthResponse struct {
	Status string `json:"status"`
}

type CreateEventRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	StartAt     string `json:"start_at"`
	EndAt       string `json:"end_at"`
}

type BusinessLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type BusinessRegisterRequest struct {
	OrganizationName string `json:"organization_name"`
	FirstName        string `json:"first_name"`
	LastName         string `json:"last_name"`
	Email            string `json:"email"`
	Password         string `json:"password"`
}

type AuthTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresAt   string `json:"expires_at"`
}

type BusinessLoginResponse struct {
	AuthTokenResponse
	User BusinessUserProfile `json:"user"`
}

// BusinessLoginOTPChallengeResponse is returned after password login; JWT is issued after OTP verify.
type BusinessLoginOTPChallengeResponse struct {
	OTPRequired  bool   `json:"otp_required"`
	ChallengeID  string `json:"challenge_id"`
	MaskedEmail  string `json:"masked_email"`
	ExpiresAt    string `json:"expires_at"`
}

type BusinessOTPVerifyRequest struct {
	ChallengeID string `json:"challenge_id"`
	Code        string `json:"code"`
}

type BusinessOTPResendRequest struct {
	ChallengeID string `json:"challenge_id"`
}

type BusinessUserProfile struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Email          string `json:"email"`
	FirstName      string `json:"first_name,omitempty"`
	LastName       string `json:"last_name,omitempty"`
	Role           string `json:"role"`
}

type GoogleOAuthRequest struct {
	IDToken   string `json:"id_token"`
	GoogleSub string `json:"google_sub"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type UserLoginResponse struct {
	AuthTokenResponse
	User UserProfile `json:"user"`
}

type UserProfile struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

type UserRegisterRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone,omitempty"`
}

type UserLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type ListUserEventJoinsResponse struct {
	Joins   []domain.UserEventJoin `json:"joins"`
	Total   int                    `json:"total"`
	Page    int                    `json:"page"`
	Limit   int                    `json:"limit"`
	HasMore bool                   `json:"has_more"`
}

type ListSubscribedGeofenceEventsResponse struct {
	Events      []domain.SubscribedGeofenceEvent `json:"events"`
	Alerts      []domain.GeofenceAlert           `json:"alerts"`
	RefreshedAt string                           `json:"refreshed_at"`
}

type ListUserActivityHistoryResponse struct {
	Sessions    []domain.UserActivitySession `json:"sessions"`
	RetentionDays int                        `json:"retention_days"`
}

type UpdateMyProfileRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type RecordClockInRequest struct {
	EventID      string  `json:"event_id"`
	EventTitle   string  `json:"event_title"`
	FenceID      string  `json:"fence_id"`
	FenceName    string  `json:"fence_name"`
	Source       string  `json:"source"`
	Lat          float64 `json:"lat"`
	Lng          float64 `json:"lng"`
	AccuracyM    float64 `json:"accuracy_m,omitempty"`
	MockLocation bool    `json:"mock_location,omitempty"`
}

type RecordClockOutRequest struct {
	SessionID string `json:"session_id,omitempty"`
	EventID   string `json:"event_id,omitempty"`
	FenceID   string `json:"fence_id,omitempty"`
}

type ListEventsResponse struct {
	Events []domain.Event `json:"events"`
}

type CreateFenceRequest struct {
	Name        string  `json:"name"`
	ShapeType   string  `json:"shape_type"`
	StartAt     string  `json:"start_at,omitempty"`
	EndAt       string  `json:"end_at,omitempty"`
	CenterLat   float64 `json:"center_lat"`
	CenterLng   float64 `json:"center_lng"`
	RadiusM     float64 `json:"radius_m"`
	PolygonJSON string  `json:"polygon_geojson"`
}

type JoinByQRRequest struct {
	UserID    string `json:"user_id,omitempty"`
	Email     string `json:"email,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	QRToken   string `json:"qr_token"`
}

type MarkPresentRequest struct {
	UserID       string  `json:"user_id"`
	Source       string  `json:"source"`
	Lat          float64 `json:"lat"`
	Lng          float64 `json:"lng"`
	AccuracyM    float64 `json:"accuracy_m"`
	MockLocation bool    `json:"mock_location,omitempty"`
}

type ConsentTemplateRequest struct {
	RequiredFields []domain.ConsentField `json:"required_fields"`
	TrackEntryExit bool                  `json:"track_entry_exit"`
	TrackMovement  bool                  `json:"track_movement"`
}

