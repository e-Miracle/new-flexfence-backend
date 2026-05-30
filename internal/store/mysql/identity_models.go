package mysql

import "time"

// OrganizationModel is the SaaS tenant (customer business).
type OrganizationModel struct {
	ID        string    `gorm:"column:id;primaryKey;size:64"`
	Name      string    `gorm:"column:name;size:255;not null"`
	Slug      string    `gorm:"column:slug;size:128;not null;uniqueIndex"`
	Plan      string    `gorm:"column:plan;size:32;not null;default:trial"`
	Status    string    `gorm:"column:status;size:32;not null;default:active;index"`
	CreatedAt time.Time `gorm:"column:created_at;not null"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null"`
}

func (OrganizationModel) TableName() string { return "organizations" }

// BusinessUserModel is a dashboard operator belonging to one organization.
type BusinessUserModel struct {
	ID             string     `gorm:"column:id;primaryKey;size:64"`
	OrganizationID string     `gorm:"column:organization_id;size:64;not null;index:idx_business_users_org_email,unique"`
	Email          string     `gorm:"column:email;size:255;not null;index:idx_business_users_org_email,unique"`
	PasswordHash   string     `gorm:"column:password_hash;size:255"`
	FirstName      string     `gorm:"column:first_name;size:128"`
	LastName       string     `gorm:"column:last_name;size:128"`
	Role           string     `gorm:"column:role;size:32;not null;default:admin"`
	Status         string     `gorm:"column:status;size:32;not null;default:active;index"`
	GoogleSub      string     `gorm:"column:google_sub;size:255;uniqueIndex"`
	LastLoginAt    *time.Time `gorm:"column:last_login_at"`
	CreatedAt      time.Time  `gorm:"column:created_at;not null"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;not null"`
}

func (BusinessUserModel) TableName() string { return "business_users" }

// UserModel is an end-user (attendee) on mobile apps.
type UserModel struct {
	ID           string    `gorm:"column:id;primaryKey;size:64"`
	Email        string    `gorm:"column:email;size:255;not null;uniqueIndex"`
	PasswordHash string    `gorm:"column:password_hash;size:255"`
	FirstName    string    `gorm:"column:first_name;size:128"`
	LastName     string    `gorm:"column:last_name;size:128"`
	Phone        string    `gorm:"column:phone;size:32"`
	Status       string    `gorm:"column:status;size:32;not null;default:active;index"`
	GoogleSub    string    `gorm:"column:google_sub;size:255;uniqueIndex"`
	AppleSub     string    `gorm:"column:apple_sub;size:255;uniqueIndex"`
	CreatedAt    time.Time `gorm:"column:created_at;not null"`
	UpdatedAt    time.Time `gorm:"column:updated_at;not null"`
}

func (UserModel) TableName() string { return "users" }
