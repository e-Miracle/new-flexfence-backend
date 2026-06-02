package mysql

import "time"

type UserConsentModel struct {
	ID                  string    `gorm:"column:id;primaryKey;size:64"`
	EventID             string    `gorm:"column:event_id;size:64;not null;uniqueIndex:idx_user_event_consent"`
	UserID              string    `gorm:"column:user_id;size:64;not null;uniqueIndex:idx_user_event_consent"`
	ConsentSnapshotJSON string    `gorm:"column:consent_snapshot_json;type:longtext;not null"`
	AgreedAt            time.Time `gorm:"column:agreed_at;not null"`
	CreatedAt           time.Time `gorm:"column:created_at;not null"`
}

func (UserConsentModel) TableName() string { return "user_consents" }
