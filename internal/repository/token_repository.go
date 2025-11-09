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

// tokenRepository implements TokenRepository interface
type tokenRepository struct {
	db *database.Postgres
}

// NewTokenRepository creates a new token repository
func NewTokenRepository(db *database.Postgres) TokenRepository {
	return &tokenRepository{db: db}
}

// Create creates a new refresh token in the database
func (r *tokenRepository) Create(ctx context.Context, token *domain.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at, device_info, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	// Generate UUID if not provided
	if token.ID == "" {
		token.ID = uuid.New().String()
	}

	now := time.Now()
	if token.CreatedAt.IsZero() {
		token.CreatedAt = now
	}

	_, err := r.db.DB.ExecContext(ctx, query,
		token.ID,
		token.UserID,
		token.TokenHash,
		token.ExpiresAt,
		token.CreatedAt,
		token.DeviceInfo,
		token.IPAddress,
	)

	if err != nil {
		// Check for unique constraint violation (duplicate token hash)
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" { // unique_violation
				return fmt.Errorf("token with hash already exists: %w", ErrDuplicateToken)
			}
		}
		return fmt.Errorf("failed to create token: %w", err)
	}

	return nil
}

// GetByTokenHash retrieves a refresh token by its hash
func (r *tokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, created_at, device_info, ip_address
		FROM refresh_tokens
		WHERE token_hash = $1
	`

	token := &domain.RefreshToken{}
	var deviceInfo, ipAddress sql.NullString

	err := r.db.DB.QueryRowContext(ctx, query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.CreatedAt,
		&deviceInfo,
		&ipAddress,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("token with hash not found: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get token by hash: %w", err)
	}

	if deviceInfo.Valid {
		token.DeviceInfo = &deviceInfo.String
	}
	if ipAddress.Valid {
		token.IPAddress = &ipAddress.String
	}

	return token, nil
}

// GetByUserID retrieves all refresh tokens for a user
func (r *tokenRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, created_at, device_info, ip_address
		FROM refresh_tokens
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tokens by user id: %w", err)
	}
	defer rows.Close()

	var tokens []*domain.RefreshToken
	for rows.Next() {
		token := &domain.RefreshToken{}
		var deviceInfo, ipAddress sql.NullString

		err := rows.Scan(
			&token.ID,
			&token.UserID,
			&token.TokenHash,
			&token.ExpiresAt,
			&token.CreatedAt,
			&deviceInfo,
			&ipAddress,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan token: %w", err)
		}

		if deviceInfo.Valid {
			token.DeviceInfo = &deviceInfo.String
		}
		if ipAddress.Valid {
			token.IPAddress = &ipAddress.String
		}

		tokens = append(tokens, token)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate tokens: %w", err)
	}

	return tokens, nil
}

// Delete deletes a refresh token by ID
func (r *tokenRepository) Delete(ctx context.Context, tokenID string) error {
	query := `DELETE FROM refresh_tokens WHERE id = $1`

	result, err := r.db.DB.ExecContext(ctx, query, tokenID)
	if err != nil {
		return fmt.Errorf("failed to delete token: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("token with id %s not found: %w", tokenID, ErrNotFound)
	}

	return nil
}

// DeleteByTokenHash deletes a refresh token by its hash
func (r *tokenRepository) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	query := `DELETE FROM refresh_tokens WHERE token_hash = $1`

	result, err := r.db.DB.ExecContext(ctx, query, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to delete token by hash: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("token with hash not found: %w", ErrNotFound)
	}

	return nil
}

// DeleteExpired deletes all expired refresh tokens
func (r *tokenRepository) DeleteExpired(ctx context.Context) error {
	query := `DELETE FROM refresh_tokens WHERE expires_at < $1`

	_, err := r.db.DB.ExecContext(ctx, query, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete expired tokens: %w", err)
	}

	return nil
}
