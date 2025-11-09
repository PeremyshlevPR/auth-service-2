package repository

import (
	"context"

	"github.com/prperemyshlev/auth-service-2/internal/domain"
)

// UserRepository defines methods for user operations
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	UpdateLastLogin(ctx context.Context, userID string) error
}

// TokenRepository defines methods for token operations
type TokenRepository interface {
	Create(ctx context.Context, token *domain.RefreshToken) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)
	GetByUserID(ctx context.Context, userID string) ([]*domain.RefreshToken, error)
	Delete(ctx context.Context, tokenID string) error
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
	DeleteExpired(ctx context.Context) error
}

// OAuthProviderRepository defines methods for OAuth provider operations
type OAuthProviderRepository interface {
	Create(ctx context.Context, provider *domain.OAuthProvider) error
	GetByProvider(ctx context.Context, provider, providerUserID string) (*domain.OAuthProvider, error)
	GetByUserID(ctx context.Context, userID string) ([]*domain.OAuthProvider, error)
	Delete(ctx context.Context, providerID string) error
}
