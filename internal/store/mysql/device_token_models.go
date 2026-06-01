package mysql

import "time"

type UserDeviceTokenModel struct {
	ID        string    `gorm:"column:id;primaryKey;size:64"`
	UserID    string    `gorm:"column:user_id;size:64;not null;index"`
	Platform  string    `gorm:"column:platform;size:16;not null"`
	Token     string    `gorm:"column:token;size:512;not null;uniqueIndex"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null"`
}

func (UserDeviceTokenModel) TableName() string { return "user_device_tokens" }
