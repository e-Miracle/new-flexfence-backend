package domain

import "time"

// Consent value types for custom attendee inputs (mobile renders matching control).
const (
	ConsentValueText    = "text"
	ConsentValueEmail   = "email"
	ConsentValuePhone   = "phone"
	ConsentValueNumber  = "number"
	ConsentValueDate    = "date"
	ConsentValueBoolean = "boolean"
)

type ConsentField struct {
	Key       string `json:"key"`
	Label     string `json:"label"`
	Required  bool   `json:"required"`
	IsCustom  bool   `json:"is_custom,omitempty"`
	ValueType string `json:"value_type,omitempty"`
}

type ConsentFieldRecommendation struct {
	Key       string `json:"key"`
	Label     string `json:"label"`
	ValueType string `json:"value_type"`
	UseCount  int    `json:"use_count"`
}

type ConsentTemplate struct {
	ID               string         `json:"id"`
	EventID          string         `json:"event_id"`
	RequiredFields   []ConsentField `json:"required_fields"`
	TrackEntryExit   bool           `json:"track_entry_exit"`
	TrackMovement    bool           `json:"track_movement"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

// UserConsent is an immutable attendee consent snapshot for an event.
type UserConsent struct {
	ID              string            `json:"id"`
	EventID         string            `json:"event_id"`
	UserID          string            `json:"user_id"`
	ConsentSnapshot map[string]string `json:"consent_snapshot"`
	AgreedAt        time.Time         `json:"agreed_at"`
}

type AttendanceWithUser struct {
	ID        string    `json:"id"`
	EventID   string    `json:"event_id"`
	FenceID   string    `json:"fence_id,omitempty"`
	UserID    string    `json:"user_id"`
	UserEmail string    `json:"user_email,omitempty"`
	UserName  string    `json:"user_name,omitempty"`
	Status    string    `json:"status"`
	Source    string    `json:"source"`
	MarkedAt  time.Time `json:"marked_at"`
	Lat       float64   `json:"lat,omitempty"`
	Lng       float64   `json:"lng,omitempty"`
	AccuracyM float64   `json:"accuracy_m,omitempty"`
}
