package domain

import "time"

// User represents a user in the system
type User struct {
	ID              string     `json:"id" db:"id"`
	Email           string     `json:"email" db:"email"`
	PasswordHash    string     `json:"-" db:"password_hash"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
	LastLoginAt     *time.Time `json:"last_login_at" db:"last_login_at"`
	IsActive        bool       `json:"is_active" db:"is_active"`
	IsEmailVerified bool       `json:"is_email_verified" db:"is_email_verified"`
}

// RefreshToken represents a refresh token in the system
type RefreshToken struct {
	ID         string    `json:"id" db:"id"`
	UserID     string    `json:"user_id" db:"user_id"`
	TokenHash  string    `json:"-" db:"token_hash"`
	ExpiresAt  time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	DeviceInfo *string   `json:"device_info" db:"device_info"`
	IPAddress  *string   `json:"ip_address" db:"ip_address"`
}

// OAuthProvider represents an OAuth provider connection for a user
type OAuthProvider struct {
	ID             string    `json:"id" db:"id"`
	UserID         string    `json:"user_id" db:"user_id"`
	Provider       string    `json:"provider" db:"provider"` // google, apple, facebook
	ProviderUserID string    `json:"provider_user_id" db:"provider_user_id"`
	Email          *string   `json:"email" db:"email"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}
