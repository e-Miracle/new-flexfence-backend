package mysql

import "time"

type EventModel struct {
	ID             string    `gorm:"column:id;primaryKey;size:64"`
	OrganizationID string    `gorm:"column:organization_id;size:64;not null;index"`
	CreatedByID    string    `gorm:"column:created_by_id;size:64;not null;index"`
	Title          string    `gorm:"column:title;size:255;not null"`
	Description string    `gorm:"column:description;type:text"`
	StartAt     time.Time `gorm:"column:start_at"`
	EndAt       time.Time `gorm:"column:end_at"`
	Status            string     `gorm:"column:status;size:32;not null;index"`
	QRToken           string     `gorm:"column:qr_token;size:64;index"`
	GoLiveProcessedAt *time.Time `gorm:"column:go_live_processed_at;index"`
	CreatedAt         time.Time  `gorm:"column:created_at;not null"`
}

func (EventModel) TableName() string { return "events" }

type FenceModel struct {
	ID          string    `gorm:"column:id;primaryKey;size:64"`
	EventID     string    `gorm:"column:event_id;size:64;not null;index"`
	Name        string    `gorm:"column:name;size:255;not null"`
	ShapeType   string    `gorm:"column:shape_type;size:32;not null"`
	StartAt     time.Time `gorm:"column:start_at"`
	EndAt       time.Time `gorm:"column:end_at"`
	CenterLat   float64   `gorm:"column:center_lat"`
	CenterLng   float64   `gorm:"column:center_lng"`
	RadiusM     float64   `gorm:"column:radius_m"`
	PolygonJSON string    `gorm:"column:polygon_json;type:longtext"`
	CreatedAt   time.Time `gorm:"column:created_at;not null"`
}

func (FenceModel) TableName() string { return "fences" }

type EventJoinModel struct {
	ID         string    `gorm:"column:id;primaryKey;size:64"`
	EventID    string    `gorm:"column:event_id;size:64;not null;index"`
	UserID     string    `gorm:"column:user_id;size:64;not null;index"`
	JoinSource string    `gorm:"column:join_source;size:32;not null"`
	QRToken    string    `gorm:"column:qr_token;size:255"`
	JoinedAt   time.Time `gorm:"column:joined_at;not null"`
}

func (EventJoinModel) TableName() string { return "event_joins" }

type AttendanceModel struct {
	ID        string    `gorm:"column:id;primaryKey;size:64"`
	EventID   string    `gorm:"column:event_id;size:64;not null;index"`
	FenceID   string    `gorm:"column:fence_id;size:64;index"`
	UserID    string    `gorm:"column:user_id;size:64;not null;index"`
	Status    string    `gorm:"column:status;size:32;not null"`
	Source    string    `gorm:"column:source;size:32;not null"`
	MarkedAt  time.Time `gorm:"column:marked_at;not null"`
	Lat       float64   `gorm:"column:lat"`
	Lng       float64   `gorm:"column:lng"`
	AccuracyM float64   `gorm:"column:accuracy_m"`
}

func (AttendanceModel) TableName() string { return "attendance_records" }

type UserActivitySessionModel struct {
	ID         string     `gorm:"column:id;primaryKey;size:64"`
	UserID     string     `gorm:"column:user_id;size:64;not null;index"`
	EventID    string     `gorm:"column:event_id;size:64;index"`
	EventTitle string     `gorm:"column:event_title;size:255"`
	FenceID    string     `gorm:"column:fence_id;size:64;index"`
	FenceName  string     `gorm:"column:fence_name;size:255;not null"`
	ClockInAt  time.Time  `gorm:"column:clock_in_at;not null;index"`
	ClockOutAt *time.Time `gorm:"column:clock_out_at;index"`
	Verified   bool       `gorm:"column:verified;not null"`
	Source     string     `gorm:"column:source;size:32;not null"`
	CreatedAt  time.Time  `gorm:"column:created_at;not null"`
}

func (UserActivitySessionModel) TableName() string { return "user_activity_sessions" }

type FenceCaptureSessionModel struct {
	ID          string    `gorm:"column:id;primaryKey;size:64"`
	EventID     string    `gorm:"column:event_id;size:64;not null;index"`
	Token       string    `gorm:"column:token;size:128;not null;uniqueIndex"`
	TargetShape string    `gorm:"column:target_shape;size:32;not null;default:polygon"`
	Status      string    `gorm:"column:status;size:32;not null;index"`
	PointsJSON  string    `gorm:"column:points_json;type:longtext"`
	ExpiresAt   time.Time `gorm:"column:expires_at;not null;index"`
	CreatedAt   time.Time `gorm:"column:created_at;not null"`
}

func (FenceCaptureSessionModel) TableName() string { return "fence_capture_sessions" }

type GeofenceAlertModel struct {
	ID          string     `gorm:"column:id;primaryKey;size:64"`
	UserID      string     `gorm:"column:user_id;size:64;not null;index"`
	EventID     string     `gorm:"column:event_id;size:64;not null;index"`
	AlertType   string     `gorm:"column:alert_type;size:32;not null"`
	EventTitle  string     `gorm:"column:event_title;size:255;not null"`
	FenceID     string     `gorm:"column:fence_id;size:64;index"`
	FenceName   string     `gorm:"column:fence_name;size:255"`
	Message     string     `gorm:"column:message;type:text;not null"`
	CreatedAt   time.Time  `gorm:"column:created_at;not null;index"`
	DeliveredAt *time.Time `gorm:"column:delivered_at;index"`
}

func (GeofenceAlertModel) TableName() string { return "user_geofence_alerts" }
