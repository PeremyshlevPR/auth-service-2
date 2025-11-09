package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/prperemyshlev/auth-service-2/internal/domain"
	"github.com/prperemyshlev/auth-service-2/pkg/database"
)

// userRepository implements UserRepository interface
type userRepository struct {
	db *database.Postgres
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *database.Postgres) UserRepository {
	return &userRepository{db: db}
}

// Create creates a new user in the database
func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, created_at, updated_at, is_active, is_email_verified)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	// Generate UUID if not provided
	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	now := time.Now()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = now
	}

	_, err := r.db.DB.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.CreatedAt,
		user.UpdatedAt,
		user.IsActive,
		user.IsEmailVerified,
	)

	if err != nil {
		// Check for unique constraint violation (duplicate email)
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" { // unique_violation
				return fmt.Errorf("user with email %s already exists: %w", user.Email, ErrDuplicateEmail)
			}
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByEmail retrieves a user by email
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, created_at, updated_at, last_login_at, is_active, is_email_verified
		FROM users
		WHERE email = $1
	`

	user := &domain.User{}
	var lastLoginAt sql.NullTime

	err := r.db.DB.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
		&lastLoginAt,
		&user.IsActive,
		&user.IsEmailVerified,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user with email %s not found: %w", email, ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}

	return user, nil
}

// GetByID retrieves a user by ID
func (r *userRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, created_at, updated_at, last_login_at, is_active, is_email_verified
		FROM users
		WHERE id = $1
	`

	user := &domain.User{}
	var lastLoginAt sql.NullTime

	err := r.db.DB.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
		&lastLoginAt,
		&user.IsActive,
		&user.IsEmailVerified,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user with id %s not found: %w", id, ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}

	return user, nil
}

// Update updates an existing user
func (r *userRepository) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users
		SET email = $2, password_hash = $3, is_active = $4, is_email_verified = $5
		WHERE id = $1
	`

	result, err := r.db.DB.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.IsActive,
		user.IsEmailVerified,
	)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" { // unique_violation
				return fmt.Errorf("user with email %s already exists: %w", user.Email, ErrDuplicateEmail)
			}
		}
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user with id %s not found: %w", user.ID, ErrNotFound)
	}

	return nil
}

// UpdateLastLogin updates the last login timestamp for a user
func (r *userRepository) UpdateLastLogin(ctx context.Context, userID string) error {
	query := `
		UPDATE users
		SET last_login_at = $1
		WHERE id = $2
	`

	result, err := r.db.DB.ExecContext(ctx, query, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user with id %s not found: %w", userID, ErrNotFound)
	}

	return nil
}
