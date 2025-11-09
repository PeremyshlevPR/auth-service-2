package repository

import "errors"

// Common repository errors
var (
	// ErrNotFound is returned when a record is not found
	ErrNotFound = errors.New("record not found")

	// ErrDuplicateEmail is returned when trying to create a user with an existing email
	ErrDuplicateEmail = errors.New("user with this email already exists")

	// ErrDuplicateToken is returned when trying to create a token with an existing hash
	ErrDuplicateToken = errors.New("token with this hash already exists")

	// ErrDuplicateOAuthProvider is returned when trying to create a duplicate OAuth provider connection
	ErrDuplicateOAuthProvider = errors.New("oauth provider connection already exists")
)
