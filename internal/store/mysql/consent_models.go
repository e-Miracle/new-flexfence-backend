package mysql

import "time"

type ConsentTemplateModel struct {
	ID                 string    `gorm:"column:id;primaryKey;size:64"`
	EventID            string    `gorm:"column:event_id;size:64;not null;uniqueIndex"`
	RequiredFieldsJSON string    `gorm:"column:required_fields_json;type:longtext;not null"`
	TrackEntryExit     bool      `gorm:"column:track_entry_exit;not null;default:false"`
	TrackMovement      bool      `gorm:"column:track_movement;not null;default:false"`
	CreatedAt          time.Time `gorm:"column:created_at;not null"`
	UpdatedAt          time.Time `gorm:"column:updated_at;not null"`
}

func (ConsentTemplateModel) TableName() string { return "consent_templates" }
