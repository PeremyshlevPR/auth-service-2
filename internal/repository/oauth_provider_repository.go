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

// oauthProviderRepository implements OAuthProviderRepository interface
type oauthProviderRepository struct {
	db *database.Postgres
}

// NewOAuthProviderRepository creates a new OAuth provider repository
func NewOAuthProviderRepository(db *database.Postgres) OAuthProviderRepository {
	return &oauthProviderRepository{db: db}
}

// Create creates a new OAuth provider connection
func (r *oauthProviderRepository) Create(ctx context.Context, provider *domain.OAuthProvider) error {
	query := `
		INSERT INTO oauth_providers (id, user_id, provider, provider_user_id, email, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	// Generate UUID if not provided
	if provider.ID == "" {
		provider.ID = uuid.New().String()
	}

	now := time.Now()
	if provider.CreatedAt.IsZero() {
		provider.CreatedAt = now
	}

	_, err := r.db.DB.ExecContext(ctx, query,
		provider.ID,
		provider.UserID,
		provider.Provider,
		provider.ProviderUserID,
		provider.Email,
		provider.CreatedAt,
	)

	if err != nil {
		// Check for unique constraint violation (duplicate provider + provider_user_id)
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" { // unique_violation
				return fmt.Errorf("oauth provider connection already exists: %w", ErrDuplicateOAuthProvider)
			}
		}
		return fmt.Errorf("failed to create oauth provider: %w", err)
	}

	return nil
}

// GetByProvider retrieves an OAuth provider connection by provider and provider user ID
func (r *oauthProviderRepository) GetByProvider(ctx context.Context, provider, providerUserID string) (*domain.OAuthProvider, error) {
	query := `
		SELECT id, user_id, provider, provider_user_id, email, created_at
		FROM oauth_providers
		WHERE provider = $1 AND provider_user_id = $2
	`

	oauthProvider := &domain.OAuthProvider{}
	var email sql.NullString

	err := r.db.DB.QueryRowContext(ctx, query, provider, providerUserID).Scan(
		&oauthProvider.ID,
		&oauthProvider.UserID,
		&oauthProvider.Provider,
		&oauthProvider.ProviderUserID,
		&email,
		&oauthProvider.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("oauth provider connection not found: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get oauth provider: %w", err)
	}

	if email.Valid {
		oauthProvider.Email = &email.String
	}

	return oauthProvider, nil
}

// GetByUserID retrieves all OAuth provider connections for a user
func (r *oauthProviderRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.OAuthProvider, error) {
	query := `
		SELECT id, user_id, provider, provider_user_id, email, created_at
		FROM oauth_providers
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get oauth providers by user id: %w", err)
	}
	defer rows.Close()

	var providers []*domain.OAuthProvider
	for rows.Next() {
		provider := &domain.OAuthProvider{}
		var email sql.NullString

		err := rows.Scan(
			&provider.ID,
			&provider.UserID,
			&provider.Provider,
			&provider.ProviderUserID,
			&email,
			&provider.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan oauth provider: %w", err)
		}

		if email.Valid {
			provider.Email = &email.String
		}

		providers = append(providers, provider)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate oauth providers: %w", err)
	}

	return providers, nil
}

// Delete deletes an OAuth provider connection by ID
func (r *oauthProviderRepository) Delete(ctx context.Context, providerID string) error {
	query := `DELETE FROM oauth_providers WHERE id = $1`

	result, err := r.db.DB.ExecContext(ctx, query, providerID)
	if err != nil {
		return fmt.Errorf("failed to delete oauth provider: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("oauth provider with id %s not found: %w", providerID, ErrNotFound)
	}

	return nil
}
