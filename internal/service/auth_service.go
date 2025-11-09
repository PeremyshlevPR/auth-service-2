package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/prperemyshlev/auth-service-2/internal/domain"
	"github.com/prperemyshlev/auth-service-2/internal/dto"
	"github.com/prperemyshlev/auth-service-2/internal/repository"
	"github.com/prperemyshlev/auth-service-2/internal/utils"
)

// authService implements AuthService interface
type authService struct {
	userRepo           repository.UserRepository
	tokenRepo          repository.TokenRepository
	jwtManager         *utils.JWTManager
	blacklistService   *TokenBlacklistService
	bcryptCost         int
	refreshTokenExpiry time.Duration
}

// NewAuthService creates a new auth service
func NewAuthService(
	userRepo repository.UserRepository,
	tokenRepo repository.TokenRepository,
	jwtManager *utils.JWTManager,
	blacklistService *TokenBlacklistService,
	bcryptCost int,
	refreshTokenExpiry time.Duration,
) AuthService {
	return &authService{
		userRepo:           userRepo,
		tokenRepo:          tokenRepo,
		jwtManager:         jwtManager,
		blacklistService:   blacklistService,
		bcryptCost:         bcryptCost,
		refreshTokenExpiry: refreshTokenExpiry,
	}
}

// Register registers a new user
func (s *authService) Register(ctx context.Context, req *dto.RegisterRequest) (*AuthResponseWithRefreshToken, error) {
	// Validate email format
	if !utils.ValidateEmail(req.Email) {
		return nil, fmt.Errorf("invalid email format")
	}

	// Validate password
	if !utils.ValidatePassword(req.Password) {
		return nil, fmt.Errorf("password must be at least 8 characters long and contain uppercase, lowercase, and number")
	}

	// Check if user already exists
	_, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err == nil {
		return nil, fmt.Errorf("user with email %s already exists", req.Email)
	}
	// If error is not NotFound, return it
	if err != repository.ErrNotFound {
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}

	// Hash password
	passwordHash, err := utils.HashPassword(req.Password, s.bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &domain.User{
		Email:           utils.SanitizeEmail(req.Email),
		PasswordHash:    passwordHash,
		IsActive:        true,
		IsEmailVerified: false,
	}

	err = s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate tokens
	return s.generateAuthResponseWithRefreshToken(ctx, user)
}

// Login authenticates a user
func (s *authService) Login(ctx context.Context, req *dto.LoginRequest) (*AuthResponseWithRefreshToken, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, utils.SanitizeEmail(req.Email))
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, fmt.Errorf("invalid email or password")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if user is active
	if !user.IsActive {
		return nil, fmt.Errorf("user account is inactive")
	}

	// Check password
	if !utils.CheckPasswordHash(req.Password, user.PasswordHash) {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Update last login
	err = s.userRepo.UpdateLastLogin(ctx, user.ID)
	if err != nil {
		// Log error but don't fail the login
		_ = err
	}

	// Generate tokens
	return s.generateAuthResponseWithRefreshToken(ctx, user)
}

// RefreshToken refreshes access and refresh tokens
func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*AuthResponseWithRefreshToken, error) {
	// Validate refresh token
	userID, err := s.jwtManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Hash the refresh token to check in database
	tokenHash := s.hashToken(refreshToken)

	// Check if token exists in database
	dbToken, err := s.tokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, fmt.Errorf("invalid refresh token")
		}
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	// Check if token is expired
	if time.Now().After(dbToken.ExpiresAt) {
		return nil, fmt.Errorf("refresh token expired")
	}

	// Check if token is blacklisted
	isBlacklisted, err := s.blacklistService.IsTokenBlacklisted(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to check token blacklist: %w", err)
	}
	if isBlacklisted {
		return nil, fmt.Errorf("refresh token is blacklisted")
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if user is active
	if !user.IsActive {
		return nil, fmt.Errorf("user account is inactive")
	}

	// Invalidate old refresh token (add to blacklist and delete from DB)
	err = s.blacklistService.AddToken(ctx, refreshToken, s.refreshTokenExpiry)
	if err != nil {
		// Log error but continue
		_ = err
	}

	err = s.tokenRepo.DeleteByTokenHash(ctx, tokenHash)
	if err != nil {
		// Log error but continue
		_ = err
	}

	// Generate new tokens
	return s.generateAuthResponseWithRefreshToken(ctx, user)
}

// Logout logs out a user
func (s *authService) Logout(ctx context.Context, userID, refreshToken string) error {
	if refreshToken != "" {
		// Hash the refresh token
		tokenHash := s.hashToken(refreshToken)

		// Check if token exists
		dbToken, err := s.tokenRepo.GetByTokenHash(ctx, tokenHash)
		if err == nil && dbToken.UserID == userID {
			// Add to blacklist
			err = s.blacklistService.AddToken(ctx, refreshToken, s.refreshTokenExpiry)
			if err != nil {
				// Log error but continue
				_ = err
			}

			// Delete from database
			err = s.tokenRepo.DeleteByTokenHash(ctx, tokenHash)
			if err != nil {
				// Log error but continue
				_ = err
			}
		}
	}

	return nil
}

// GetUser gets user information
func (s *authService) GetUser(ctx context.Context, userID string) (*dto.UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	response := &dto.UserResponse{
		ID:              user.ID,
		Email:           user.Email,
		CreatedAt:       user.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       user.UpdatedAt.Format(time.RFC3339),
		IsEmailVerified: user.IsEmailVerified,
	}

	if user.LastLoginAt != nil {
		lastLogin := user.LastLoginAt.Format(time.RFC3339)
		response.LastLoginAt = &lastLogin
	}

	return response, nil
}

// ValidateToken validates an access token
func (s *authService) ValidateToken(ctx context.Context, token string) (*domain.TokenClaims, error) {
	// Check if token is blacklisted
	isBlacklisted, err := s.blacklistService.IsTokenBlacklisted(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to check token blacklist: %w", err)
	}
	if isBlacklisted {
		return nil, fmt.Errorf("token is blacklisted")
	}

	// Validate token
	claims, err := s.jwtManager.ValidateToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	return claims, nil
}

// hashToken hashes a token using SHA256
func (s *authService) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
