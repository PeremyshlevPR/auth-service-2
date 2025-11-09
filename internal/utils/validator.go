package utils

import (
	"regexp"
	"strings"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// ValidateEmail validates an email address
func ValidateEmail(email string) bool {
	return emailRegex.MatchString(email)
}

// ValidatePassword validates a password
// Minimum 8 characters, at least one uppercase letter, one lowercase letter, one number
func ValidatePassword(password string) bool {
	if len(password) < 8 {
		return false
	}

	hasUpper := false
	hasLower := false
	hasNumber := false

	for _, char := range password {
		switch {
		case 'A' <= char && char <= 'Z':
			hasUpper = true
		case 'a' <= char && char <= 'z':
			hasLower = true
		case '0' <= char && char <= '9':
			hasNumber = true
		}
	}

	return hasUpper && hasLower && hasNumber
}

// SanitizeEmail sanitizes an email address
func SanitizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
