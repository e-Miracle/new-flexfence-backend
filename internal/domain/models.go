package domain

import "time"

type Event struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	CreatedByID    string    `json:"created_by_id"`
	Title          string    `json:"title"`
	Description    string    `json:"description,omitempty"`
	StartAt        time.Time `json:"start_at,omitempty"`
	EndAt          time.Time `json:"end_at,omitempty"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	QRToken        string    `json:"-"` // never exposed on generic event APIs
	ScanToClockInEnabled     bool       `json:"-"`
	ClockInQRToken           string     `json:"-"`
	ClockInQRIssuedAt        *time.Time `json:"-"`
	ClockInQRRotationMinutes int        `json:"-"`
}

// EventShare is returned to organizers for invite links and QR encoding.
type EventShare struct {
	EventID        string `json:"event_id"`
	EventTitle     string `json:"event_title"`
	QRToken        string `json:"qr_token"`
	JoinDeepLink   string `json:"join_deep_link"`
	JoinWebLink    string `json:"join_web_link,omitempty"`
	QRCodePayload  string `json:"qr_code_payload"`
}

// EventClockInShare is returned to organizers for scan-to-clock-in QR encoding.
type EventClockInShare struct {
	EventID                 string     `json:"event_id"`
	EventTitle              string     `json:"event_title"`
	ScanToClockInEnabled    bool       `json:"scan_to_clock_in_enabled"`
	QRToken                 string     `json:"qr_token,omitempty"`
	IssuedAt                *time.Time `json:"issued_at,omitempty"`
	ExpiresAt               *time.Time `json:"expires_at,omitempty"`
	RotationIntervalMinutes int        `json:"rotation_interval_minutes"`
	ClockInDeepLink         string     `json:"clock_in_deep_link"`
	ClockInWebLink          string     `json:"clock_in_web_link,omitempty"`
	QRCodePayload           string     `json:"qr_code_payload"`
}

type Fence struct {
	ID          string    `json:"id"`
	EventID     string    `json:"event_id"`
	Name        string    `json:"name"`
	ShapeType   string    `json:"shape_type"`
	StartAt     time.Time `json:"start_at"`
	EndAt       time.Time `json:"end_at"`
	CenterLat   float64   `json:"center_lat,omitempty"`
	CenterLng   float64   `json:"center_lng,omitempty"`
	RadiusM     float64   `json:"radius_m,omitempty"`
	PolygonJSON string    `json:"polygon_geojson,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// FenceCreateInput is resolved input for persisting a fence.
type FenceCreateInput struct {
	Name        string
	ShapeType   string
	StartAt     time.Time
	EndAt       time.Time
	CenterLat   float64
	CenterLng   float64
	RadiusM     float64
	PolygonJSON string
}

type EventJoin struct {
	ID         string    `json:"id"`
	EventID    string    `json:"event_id"`
	UserID     string    `json:"user_id"`
	JoinSource string    `json:"join_source"`
	QRToken    string    `json:"qr_token,omitempty"`
	JoinedAt   time.Time `json:"joined_at"`
}

// UserEventJoin is an event the attendee joined, with display fields for mobile.
type UserEventJoin struct {
	ID               string    `json:"id"`
	EventID          string    `json:"event_id"`
	EventTitle       string    `json:"event_title"`
	EventDescription string    `json:"event_description,omitempty"`
	EventStartAt     time.Time `json:"event_start_at,omitempty"`
	EventEndAt       time.Time `json:"event_end_at,omitempty"`
	EventStatus      string    `json:"event_status,omitempty"`
	JoinSource       string    `json:"join_source"`
	JoinedAt         time.Time `json:"joined_at"`
}

// GeofenceAlert is a mobile notification payload delivered via subscribed-events sync.
type GeofenceAlert struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	EventID    string    `json:"event_id"`
	EventTitle string    `json:"event_title"`
	FenceID    string    `json:"fence_id,omitempty"`
	FenceName  string    `json:"fence_name,omitempty"`
	Message    string    `json:"message"`
	CreatedAt  time.Time `json:"created_at"`
}

// SubscribedGeofenceEvent is a joined event with circle fence geometry for mobile geofencing.
type SubscribedGeofenceEvent struct {
	ID               string    `json:"id"`
	EventID          string    `json:"event_id"`
	EventTitle       string    `json:"event_title"`
	EventDescription string    `json:"event_description,omitempty"`
	EventStartAt     time.Time `json:"event_start_at,omitempty"`
	EventEndAt       time.Time `json:"event_end_at,omitempty"`
	EventStatus      string    `json:"event_status,omitempty"`
	JoinSource            string    `json:"join_source"`
	JoinedAt              time.Time `json:"joined_at"`
	ScanToClockInEnabled  bool      `json:"scan_to_clock_in_enabled"`
	Fences                []Fence   `json:"fences"`
}

type AttendanceRecord struct {
	ID        string    `json:"id"`
	EventID   string    `json:"event_id"`
	FenceID   string    `json:"fence_id,omitempty"`
	UserID    string    `json:"user_id"`
	Status    string    `json:"status"`
	Source    string    `json:"source"`
	MarkedAt  time.Time `json:"marked_at"`
	Lat       float64   `json:"lat,omitempty"`
	Lng       float64   `json:"lng,omitempty"`
	AccuracyM float64   `json:"accuracy_m,omitempty"`
}

// UserActivitySession is a clock-in/out session shown in mobile history.
type UserActivitySession struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	EventID    string     `json:"event_id,omitempty"`
	EventTitle string     `json:"event_title,omitempty"`
	FenceID    string     `json:"fence_id,omitempty"`
	FenceName  string     `json:"fence_name"`
	ClockInAt  time.Time  `json:"clock_in_at"`
	ClockOutAt *time.Time `json:"clock_out_at,omitempty"`
	Verified   bool       `json:"verified"`
	Source     string     `json:"source"`
	CreatedAt  time.Time  `json:"created_at"`
}

// FenceCapturePoint is a GPS sample submitted from mobile.
type FenceCapturePoint struct {
	Lat        float64   `json:"lat"`
	Lng        float64   `json:"lng"`
	AccuracyM  float64   `json:"accuracy_m,omitempty"`
	Role       string    `json:"role,omitempty"` // center, edge (circle); empty for polygon vertices
	CapturedAt time.Time `json:"captured_at"`
}

// FenceCaptureSession lets an organizer collect fence coordinates via a one-time mobile link.
type FenceCaptureSession struct {
	ID          string              `json:"id"`
	EventID     string              `json:"event_id"`
	EventTitle  string              `json:"event_title,omitempty"`
	Token       string              `json:"token"`
	TargetShape string              `json:"target_shape"` // circle or polygon
	Status      string              `json:"status"`
	Points      []FenceCapturePoint `json:"points"`
	ExpiresAt   time.Time           `json:"expires_at"`
	CreatedAt   time.Time           `json:"created_at"`
}
