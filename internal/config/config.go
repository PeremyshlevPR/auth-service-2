package config

import (
	"context"
	"fmt"

	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	Server   ServerConfig   `env:",prefix=SERVER_"`
	Postgres PostgresConfig `env:",prefix=POSTGRES_"`
	Redis    RedisConfig    `env:",prefix=REDIS_"`
	JWT      JWTConfig      `env:",prefix=JWT_"`
	Security SecurityConfig `env:",prefix="`
	CORS     CORSConfig     `env:",prefix=CORS_"`
	Env      string         `env:"ENV,default=development"`
}

type ServerConfig struct {
	Port         string   `env:"PORT,default=8080"`
	Host         string   `env:"HOST,default=0.0.0.0"`
	ReadTimeout  Duration `env:"READ_TIMEOUT,default=15s"`
	WriteTimeout Duration `env:"WRITE_TIMEOUT,default=15s"`
}

type PostgresConfig struct {
	Host     string `env:"HOST,default=localhost"`
	Port     string `env:"PORT,default=5432"`
	User     string `env:"USER,default=auth_service"`
	Password string `env:"PASSWORD,default=auth_service_password"`
	DBName   string `env:"DB,default=auth_service_db"`
	SSLMode  string `env:"SSLMODE,default=disable"`
}

type RedisConfig struct {
	Host     string `env:"HOST,default=localhost"`
	Port     string `env:"PORT,default=6379"`
	Password string `env:"PASSWORD,default="`
	DB       int    `env:"DB,default=0"`
}

type JWTConfig struct {
	Secret             string   `env:"SECRET,required"`
	AccessTokenExpiry  Duration `env:"ACCESS_TOKEN_EXPIRY,default=15m"`
	RefreshTokenExpiry Duration `env:"REFRESH_TOKEN_EXPIRY,default=7d"`
}

type SecurityConfig struct {
	BCryptCost        int      `env:"BCRYPT_COST,default=12"`
	RateLimitRequests int      `env:"RATE_LIMIT_REQUESTS,default=10"`
	RateLimitWindow   Duration `env:"RATE_LIMIT_WINDOW,default=1m"`
}

type CORSConfig struct {
	AllowedOrigins []string `env:"ALLOWED_ORIGINS,default=http://localhost:3000"`
	AllowedMethods []string `env:"ALLOWED_METHODS,default=GET,POST,PUT,DELETE,OPTIONS"`
	AllowedHeaders []string `env:"ALLOWED_HEADERS,default=Content-Type,Authorization"`
}

// DSN returns PostgreSQL connection string
func (p PostgresConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		p.Host, p.Port, p.User, p.Password, p.DBName, p.SSLMode)
}

// Address returns Redis connection address
func (r RedisConfig) Address() string {
	return fmt.Sprintf("%s:%s", r.Host, r.Port)
}

// Load loads configuration from environment variables
func Load(ctx context.Context) (*Config, error) {
	var config Config

	if err := envconfig.Process(ctx, &config); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate JWT secret length
	if len(config.JWT.Secret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters long")
	}

	return &config, nil
}

// LoadWithDefaults loads configuration with default context
func LoadWithDefaults() (*Config, error) {
	return Load(context.Background())
}
