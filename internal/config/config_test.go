package config

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Set required environment variable
	os.Setenv("JWT_SECRET", "test-secret-key-that-is-at-least-32-characters-long")
	defer os.Unsetenv("JWT_SECRET")

	ctx := context.Background()
	cfg, err := Load(ctx)
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	// Test default values
	if cfg.Server.Port != "8080" {
		t.Errorf("Expected Server.Port to be '8080', got '%s'", cfg.Server.Port)
	}

	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Expected Server.Host to be '0.0.0.0', got '%s'", cfg.Server.Host)
	}

	if cfg.Server.ReadTimeout.Duration != 15*time.Second {
		t.Errorf("Expected Server.ReadTimeout to be 15s, got %v", cfg.Server.ReadTimeout.Duration)
	}

	if cfg.Postgres.Host != "localhost" {
		t.Errorf("Expected Postgres.Host to be 'localhost', got '%s'", cfg.Postgres.Host)
	}

	if cfg.Redis.Host != "localhost" {
		t.Errorf("Expected Redis.Host to be 'localhost', got '%s'", cfg.Redis.Host)
	}

	if cfg.JWT.AccessTokenExpiry.Duration != 15*time.Minute {
		t.Errorf("Expected JWT.AccessTokenExpiry to be 15m, got %v", cfg.JWT.AccessTokenExpiry.Duration)
	}

	if cfg.JWT.RefreshTokenExpiry.Duration != 7*24*time.Hour {
		t.Errorf("Expected JWT.RefreshTokenExpiry to be 7d, got %v", cfg.JWT.RefreshTokenExpiry.Duration)
	}

	if cfg.Security.BCryptCost != 12 {
		t.Errorf("Expected Security.BCryptCost to be 12, got %d", cfg.Security.BCryptCost)
	}

	if cfg.Env != "development" {
		t.Errorf("Expected Env to be 'development', got '%s'", cfg.Env)
	}

	// Test CORS defaults
	if len(cfg.CORS.AllowedOrigins) == 0 {
		t.Error("Expected CORS.AllowedOrigins to have at least one value")
	}

	if len(cfg.CORS.AllowedMethods) == 0 {
		t.Error("Expected CORS.AllowedMethods to have at least one value")
	}
}

func TestLoadWithCustomValues(t *testing.T) {
	// Set custom environment variables
	os.Setenv("JWT_SECRET", "test-secret-key-that-is-at-least-32-characters-long")
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("SERVER_HOST", "127.0.0.1")
	os.Setenv("POSTGRES_HOST", "postgres.example.com")
	os.Setenv("JWT_ACCESS_TOKEN_EXPIRY", "30m")
	os.Setenv("ENV", "production")
	defer func() {
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("SERVER_HOST")
		os.Unsetenv("POSTGRES_HOST")
		os.Unsetenv("JWT_ACCESS_TOKEN_EXPIRY")
		os.Unsetenv("ENV")
	}()

	ctx := context.Background()
	cfg, err := Load(ctx)
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	if cfg.Server.Port != "9090" {
		t.Errorf("Expected Server.Port to be '9090', got '%s'", cfg.Server.Port)
	}

	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Expected Server.Host to be '127.0.0.1', got '%s'", cfg.Server.Host)
	}

	if cfg.Postgres.Host != "postgres.example.com" {
		t.Errorf("Expected Postgres.Host to be 'postgres.example.com', got '%s'", cfg.Postgres.Host)
	}

	if cfg.JWT.AccessTokenExpiry.Duration != 30*time.Minute {
		t.Errorf("Expected JWT.AccessTokenExpiry to be 30m, got %v", cfg.JWT.AccessTokenExpiry.Duration)
	}

	if cfg.Env != "production" {
		t.Errorf("Expected Env to be 'production', got '%s'", cfg.Env)
	}
}

func TestLoadWithoutJWTSecret(t *testing.T) {
	// Make sure JWT_SECRET is not set
	os.Unsetenv("JWT_SECRET")

	ctx := context.Background()
	_, err := Load(ctx)
	if err == nil {
		t.Error("Expected error when JWT_SECRET is not set")
	}
}

func TestLoadWithShortJWTSecret(t *testing.T) {
	// Set JWT_SECRET that is too short
	os.Setenv("JWT_SECRET", "short")
	defer os.Unsetenv("JWT_SECRET")

	ctx := context.Background()
	_, err := Load(ctx)
	if err == nil {
		t.Error("Expected error when JWT_SECRET is too short")
	}
}

func TestPostgresDSN(t *testing.T) {
	pg := PostgresConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "test_user",
		Password: "test_password",
		DBName:   "test_db",
		SSLMode:  "disable",
	}

	dsn := pg.DSN()
	expected := "host=localhost port=5432 user=test_user password=test_password dbname=test_db sslmode=disable"
	if dsn != expected {
		t.Errorf("Expected DSN to be '%s', got '%s'", expected, dsn)
	}
}

func TestRedisAddress(t *testing.T) {
	redis := RedisConfig{
		Host: "localhost",
		Port: "6379",
	}

	addr := redis.Address()
	expected := "localhost:6379"
	if addr != expected {
		t.Errorf("Expected Address to be '%s', got '%s'", expected, addr)
	}
}
