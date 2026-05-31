package domain

import "time"

// Organization is the SaaS tenant (customer business).
type Organization struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Plan      string    `json:"plan"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BusinessUser logs into the dashboard and manages events for one organization.
type BusinessUser struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organization_id"`
	Email          string     `json:"email"`
	FirstName      string     `json:"first_name,omitempty"`
	LastName       string     `json:"last_name,omitempty"`
	Role           string     `json:"role"`
	Status         string     `json:"status"`
	GoogleSub      string     `json:"-"`
	LastLoginAt    *time.Time `json:"last_login_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// User is an end-user (attendee) on mobile apps.
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name,omitempty"`
	LastName  string    `json:"last_name,omitempty"`
	Phone         string    `json:"phone,omitempty"`
	EmailVerified bool      `json:"email_verified"`
	Status        string    `json:"status"`
	GoogleSub string    `json:"-"`
	AppleSub  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Business role constants.
const (
	BusinessRoleOwner  = "owner"
	BusinessRoleAdmin  = "admin"
	BusinessRoleViewer = "viewer"
)

// Account status constants (shared pattern).
const (
	StatusActive   = "active"
	StatusInvited  = "invited"
	StatusDisabled = "disabled"
)
