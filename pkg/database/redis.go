package database

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Redis represents a Redis client
type Redis struct {
	Client *redis.Client
}

// NewRedis creates a new Redis client
func NewRedis(addr, password string, db int) (*Redis, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &Redis{Client: client}, nil
}

// Close closes the Redis connection
func (r *Redis) Close() error {
	return r.Client.Close()
}

// Ping checks if Redis is available
func (r *Redis) Ping(ctx context.Context) error {
	return r.Client.Ping(ctx).Err()
}
