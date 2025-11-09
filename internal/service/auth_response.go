package service

import (
	"context"
	"fmt"
	"time"

	"github.com/prperemyshlev/auth-service-2/internal/domain"
	"github.com/prperemyshlev/auth-service-2/internal/dto"
)

// AuthResponseWithRefreshToken contains auth response and refresh token
type AuthResponseWithRefreshToken struct {
	AuthResponse *dto.AuthResponse
	RefreshToken string
	ExpiresIn    int // Refresh token expiry in seconds
}

// generateAuthResponseWithRefreshToken generates access and refresh tokens and returns auth response with refresh token
func (s *authService) generateAuthResponseWithRefreshToken(ctx context.Context, user *domain.User) (*AuthResponseWithRefreshToken, error) {
	// Generate access token
	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := s.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Hash refresh token for storage
	tokenHash := s.hashToken(refreshToken)

	// Save refresh token to database
	refreshTokenEntity := &domain.RefreshToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(s.refreshTokenExpiry),
	}

	err = s.tokenRepo.Create(ctx, refreshTokenEntity)
	if err != nil {
		return nil, fmt.Errorf("failed to save refresh token: %w", err)
	}

	return &AuthResponseWithRefreshToken{
		AuthResponse: &dto.AuthResponse{
			AccessToken: accessToken,
			TokenType:   "Bearer",
			ExpiresIn:   s.jwtManager.GetAccessTokenExpiry(),
			User: dto.UserInfo{
				ID:    user.ID,
				Email: user.Email,
			},
		},
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.refreshTokenExpiry.Seconds()),
	}, nil
}
