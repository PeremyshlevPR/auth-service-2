package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/prperemyshlev/auth-service-2/internal/dto"
	"github.com/prperemyshlev/auth-service-2/internal/service"
)

// AuthHandler handles authentication requests
type AuthHandler struct {
	authService service.AuthService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Register handles user registration
// @Summary Register a new user
// @Description Register a new user in the system
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.RegisterRequest true "Registration request"
// @Success 201 {object} dto.AuthResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Validation failed",
			Message: err.Error(),
		})
		return
	}

	response, err := h.authService.Register(c.Request.Context(), &req)
	if err != nil {
		// Check if user already exists
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, dto.ErrorResponse{
				Error:   "Conflict",
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad request",
			Message: err.Error(),
		})
		return
	}

	// Set refresh token in httpOnly cookie
	c.SetCookie("refresh_token", response.RefreshToken, response.ExpiresIn, "/api/v1/auth/refresh", "", true, true)

	c.JSON(http.StatusCreated, response.AuthResponse)
}

// Login handles user login
// @Summary Login user
// @Description Authenticate user with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "Login request"
// @Success 200 {object} dto.AuthResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Validation failed",
			Message: err.Error(),
		})
		return
	}

	response, err := h.authService.Login(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: err.Error(),
		})
		return
	}

	// Set refresh token in httpOnly cookie
	c.SetCookie("refresh_token", response.RefreshToken, response.ExpiresIn, "/api/v1/auth/refresh", "", true, true)

	c.JSON(http.StatusOK, response.AuthResponse)
}

// Refresh handles token refresh
// @Summary Refresh tokens
// @Description Refresh access and refresh tokens
// @Tags auth
// @Produce json
// @Success 200 {object} dto.AuthResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad request",
			Message: "Refresh token not found in cookie",
		})
		return
	}

	response, err := h.authService.RefreshToken(c.Request.Context(), refreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: err.Error(),
		})
		return
	}

	// Set new refresh token in httpOnly cookie
	c.SetCookie("refresh_token", response.RefreshToken, response.ExpiresIn, "/api/v1/auth/refresh", "", true, true)

	c.JSON(http.StatusOK, response.AuthResponse)
}

// Logout handles user logout
// @Summary Logout user
// @Description Logout user and invalidate refresh token
// @Tags auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} dto.SuccessResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "User ID not found in context",
		})
		return
	}

	refreshToken, _ := c.Cookie("refresh_token")

	err := h.authService.Logout(c.Request.Context(), userID.(string), refreshToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	// Clear refresh token cookie
	c.SetCookie("refresh_token", "", -1, "/api/v1/auth/refresh", "", true, true)

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Message: "Logged out successfully",
	})
}

// GetMe handles getting current user profile
// @Summary Get current user profile
// @Description Get information about the current authenticated user
// @Tags auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} dto.UserResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /auth/me [get]
func (h *AuthHandler) GetMe(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "User ID not found in context",
		})
		return
	}

	user, err := h.authService.GetUser(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, user)
}
