package mysql

import "time"

type OrganizationConsentFieldModel struct {
	ID             string    `gorm:"column:id;primaryKey;size:64"`
	OrganizationID string    `gorm:"column:organization_id;size:64;not null;uniqueIndex:idx_org_consent_field_key"`
	FieldKey       string    `gorm:"column:field_key;size:128;not null;uniqueIndex:idx_org_consent_field_key"`
	Label          string    `gorm:"column:label;size:255;not null"`
	ValueType      string    `gorm:"column:value_type;size:32;not null"`
	UseCount       int       `gorm:"column:use_count;not null;default:0"`
	LastUsedAt     time.Time `gorm:"column:last_used_at;not null"`
	CreatedAt      time.Time `gorm:"column:created_at;not null"`
	UpdatedAt      time.Time `gorm:"column:updated_at;not null"`
}

func (OrganizationConsentFieldModel) TableName() string {
	return "organization_consent_fields"
}
