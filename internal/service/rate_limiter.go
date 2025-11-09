package service

import (
	"context"
	"fmt"
	"time"

	"github.com/prperemyshlev/auth-service-2/pkg/database"
	"github.com/redis/go-redis/v9"
)

// RateLimiter handles rate limiting using Redis
type RateLimiter struct {
	redis *database.Redis
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(redis *database.Redis) *RateLimiter {
	return &RateLimiter{redis: redis}
}

// Allow checks if a request is allowed based on rate limit
// Returns true if request is allowed, false if rate limit exceeded
func (r *RateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	now := time.Now()
	windowStart := now.Add(-window)

	// Use sliding window log algorithm
	// Key format: "ratelimit:{key}"
	redisKey := fmt.Sprintf("ratelimit:%s", key)

	// Remove entries older than the window
	err := r.redis.Client.ZRemRangeByScore(ctx, redisKey, "0", fmt.Sprintf("%d", windowStart.Unix())).Err()
	if err != nil {
		return false, fmt.Errorf("failed to clean old entries: %w", err)
	}

	// Count current entries in the window
	count, err := r.redis.Client.ZCard(ctx, redisKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to count entries: %w", err)
	}

	// Check if limit is exceeded
	if count >= int64(limit) {
		// Get the oldest entry to calculate time until next request is allowed
		oldest, err := r.redis.Client.ZRangeWithScores(ctx, redisKey, 0, 0).Result()
		if err == nil && len(oldest) > 0 {
			// Calculate remaining time
			oldestTime := time.Unix(int64(oldest[0].Score), 0)
			remaining := window - time.Since(oldestTime)
			return false, fmt.Errorf("rate limit exceeded, try again in %v", remaining.Round(time.Second))
		}
		return false, fmt.Errorf("rate limit exceeded")
	}

	// Add current request to the set with current timestamp as score
	member := fmt.Sprintf("%d-%d", now.UnixNano(), now.Unix())
	err = r.redis.Client.ZAdd(ctx, redisKey, redis.Z{
		Score:  float64(now.Unix()),
		Member: member,
	}).Err()
	if err != nil {
		return false, fmt.Errorf("failed to add entry: %w", err)
	}

	// Set expiration on the key (window duration + 1 minute buffer)
	err = r.redis.Client.Expire(ctx, redisKey, window+time.Minute).Err()
	if err != nil {
		// Log error but don't fail the request
		_ = err
	}

	return true, nil
}

// GetRemainingRequests returns the number of remaining requests allowed
func (r *RateLimiter) GetRemainingRequests(ctx context.Context, key string, limit int, window time.Duration) (int, error) {
	now := time.Now()
	windowStart := now.Add(-window)

	redisKey := fmt.Sprintf("ratelimit:%s", key)

	// Remove entries older than the window
	err := r.redis.Client.ZRemRangeByScore(ctx, redisKey, "0", fmt.Sprintf("%d", windowStart.Unix())).Err()
	if err != nil {
		return 0, fmt.Errorf("failed to clean old entries: %w", err)
	}

	// Count current entries in the window
	count, err := r.redis.Client.ZCard(ctx, redisKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to count entries: %w", err)
	}

	remaining := limit - int(count)
	if remaining < 0 {
		remaining = 0
	}

	return remaining, nil
}
