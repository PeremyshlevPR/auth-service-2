package service

import (
	"context"
	"fmt"
	"time"

	"github.com/prperemyshlev/auth-service-2/pkg/database"
)

// TokenBlacklistService handles token blacklist operations in Redis
type TokenBlacklistService struct {
	redis *database.Redis
}

// NewTokenBlacklistService creates a new token blacklist service
func NewTokenBlacklistService(redis *database.Redis) *TokenBlacklistService {
	return &TokenBlacklistService{redis: redis}
}

// AddToken adds a token to the blacklist
func (s *TokenBlacklistService) AddToken(ctx context.Context, token string, expiry time.Duration) error {
	key := fmt.Sprintf("blacklist:token:%s", token)
	err := s.redis.Client.Set(ctx, key, "1", expiry).Err()
	if err != nil {
		return fmt.Errorf("failed to add token to blacklist: %w", err)
	}
	return nil
}

// IsTokenBlacklisted checks if a token is in the blacklist
func (s *TokenBlacklistService) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	key := fmt.Sprintf("blacklist:token:%s", token)
	exists, err := s.redis.Client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check token blacklist: %w", err)
	}
	return exists > 0, nil
}

// RemoveToken removes a token from the blacklist (if needed)
func (s *TokenBlacklistService) RemoveToken(ctx context.Context, token string) error {
	key := fmt.Sprintf("blacklist:token:%s", token)
	err := s.redis.Client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to remove token from blacklist: %w", err)
	}
	return nil
}
