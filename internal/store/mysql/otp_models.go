package mysql

import "time"

type BusinessOTPChallengeModel struct {
	ID             string     `gorm:"column:id;primaryKey;size:64"`
	BusinessUserID string     `gorm:"column:business_user_id;size:64;not null;index"`
	CodeHash       string     `gorm:"column:code_hash;size:255;not null"`
	ExpiresAt      time.Time  `gorm:"column:expires_at;not null;index"`
	Attempts       int        `gorm:"column:attempts;not null;default:0"`
	ConsumedAt     *time.Time `gorm:"column:consumed_at"`
	CreatedAt      time.Time  `gorm:"column:created_at;not null"`
}

func (BusinessOTPChallengeModel) TableName() string { return "business_otp_challenges" }
