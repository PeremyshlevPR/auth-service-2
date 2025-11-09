package dto

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email" validate:"required,email"`
	Password string `json:"password" binding:"required,min=8" validate:"required,min=8"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" validate:"required,email"`
	Password string `json:"password" binding:"required" validate:"required"`
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	AccessToken string   `json:"access_token"`
	TokenType   string   `json:"token_type"`
	ExpiresIn   int      `json:"expires_in"`
	User        UserInfo `json:"user"`
}

// UserInfo represents user information in response
type UserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// UserResponse represents a user response
type UserResponse struct {
	ID              string  `json:"id"`
	Email           string  `json:"email"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
	LastLoginAt     *string `json:"last_login_at"`
	IsEmailVerified bool    `json:"is_email_verified"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Message string `json:"message"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string      `json:"error"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}
