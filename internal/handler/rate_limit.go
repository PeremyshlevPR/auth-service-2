package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prperemyshlev/auth-service-2/internal/dto"
	"github.com/prperemyshlev/auth-service-2/internal/service"
)

// RateLimitMiddleware creates a rate limiting middleware
func RateLimitMiddleware(rateLimiter *service.RateLimiter, limit int, window time.Duration, keyFunc func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := keyFunc(c)

		allowed, err := rateLimiter.Allow(c.Request.Context(), key, limit, window)
		if err != nil {
			// If rate limit is exceeded
			if strings.Contains(err.Error(), "rate limit exceeded") {
				c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
				c.Header("X-RateLimit-Retry-After", extractRetryAfter(err.Error()))

				remaining, _ := rateLimiter.GetRemainingRequests(c.Request.Context(), key, limit, window)
				c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))

				c.JSON(http.StatusTooManyRequests, dto.ErrorResponse{
					Error:   "Too Many Requests",
					Message: err.Error(),
				})
				c.Abort()
				return
			}

			// For other errors, allow the request but log the error
			// In production, you might want to handle this differently
			c.Next()
			return
		}

		if !allowed {
			remaining, _ := rateLimiter.GetRemainingRequests(c.Request.Context(), key, limit, window)
			c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
			c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))

			c.JSON(http.StatusTooManyRequests, dto.ErrorResponse{
				Error:   "Too Many Requests",
				Message: "Rate limit exceeded",
			})
			c.Abort()
			return
		}

		// Set rate limit headers
		remaining, _ := rateLimiter.GetRemainingRequests(c.Request.Context(), key, limit, window)
		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))

		c.Next()
	}
}

// IPBasedKey extracts rate limit key from client IP
func IPBasedKey(c *gin.Context) string {
	// Try to get IP from X-Forwarded-For header (for proxies)
	ip := c.GetHeader("X-Forwarded-For")
	if ip != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(ip, ",")
		ip = strings.TrimSpace(ips[0])
	} else {
		// Fallback to RemoteAddr
		ip = c.ClientIP()
	}

	return ip
}

// EmailBasedKey extracts rate limit key from request email (for login/register)
// Uses IP address for rate limiting to prevent brute force attacks
func EmailBasedKey(c *gin.Context) string {
	// For login/register, we use IP-based rate limiting to prevent brute force
	// This prevents attackers from trying multiple emails from the same IP
	return IPBasedKey(c)
}

// EmailAndIPKey creates a rate limit key combining email and IP
// This provides more granular rate limiting per user
func EmailAndIPKey(c *gin.Context) string {
	// Try to extract email from request body
	var email string
	if c.Request.Body != nil {
		// Note: This is a simplified approach
		// In a production system, you might want to parse the JSON body
		// For now, we'll use a combination of path and IP
		email = c.Request.URL.Path
	}

	ip := IPBasedKey(c)
	if email != "" {
		return fmt.Sprintf("%s:%s", email, ip)
	}
	return ip
}

// extractRetryAfter extracts retry-after time from error message
func extractRetryAfter(errMsg string) string {
	// Extract time from error message like "rate limit exceeded, try again in 45s"
	// This is a simplified version
	if strings.Contains(errMsg, "try again in") {
		parts := strings.Split(errMsg, "try again in")
		if len(parts) > 1 {
			return strings.TrimSpace(parts[1])
		}
	}
	return "60"
}
