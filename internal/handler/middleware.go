package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/prperemyshlev/auth-service-2/internal/dto"
	"github.com/prperemyshlev/auth-service-2/internal/service"
)

// AuthMiddleware validates JWT token and adds user info to context
func AuthMiddleware(authService service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Error:   "Unauthorized",
				Message: "Authorization header is required",
			})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Error:   "Unauthorized",
				Message: "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		token := parts[1]

		// Validate token
		claims, err := authService.ValidateToken(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Error:   "Unauthorized",
				Message: "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Add user info to context
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("claims", claims)

		c.Next()
	}
}
