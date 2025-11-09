package domain

import "time"

// TokenClaims represents JWT token claims
type TokenClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Exp    int64  `json:"exp"`
	Iat    int64  `json:"iat"`
}

// TokenPair represents a pair of access and refresh tokens
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// IsExpired checks if the token is expired
func (tc TokenClaims) IsExpired() bool {
	return time.Now().Unix() > tc.Exp
}
