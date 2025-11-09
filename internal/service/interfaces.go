package service

import (
	"context"

	"github.com/prperemyshlev/auth-service-2/internal/domain"
	"github.com/prperemyshlev/auth-service-2/internal/dto"
)

// AuthService defines methods for authentication operations
type AuthService interface {
	Register(ctx context.Context, req *dto.RegisterRequest) (*AuthResponseWithRefreshToken, error)
	Login(ctx context.Context, req *dto.LoginRequest) (*AuthResponseWithRefreshToken, error)
	RefreshToken(ctx context.Context, refreshToken string) (*AuthResponseWithRefreshToken, error)
	Logout(ctx context.Context, userID, refreshToken string) error
	GetUser(ctx context.Context, userID string) (*dto.UserResponse, error)
	ValidateToken(ctx context.Context, token string) (*domain.TokenClaims, error)
}
