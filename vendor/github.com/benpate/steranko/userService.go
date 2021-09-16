package steranko

import "github.com/golang-jwt/jwt/v4"

// UserService wraps all of the functions that must be provided to Steranko by your application.
// This API matches the presto.Service API very closely, so it should be possible to wrap an
// existing presto service to serve Steranko, too.
type UserService interface {

	// New creates a newly initialized User that is ready to use
	New() User

	// Load retrieves a single User from the database
	Load(username string, user User) error

	// Save inserts/updates a single User in the database
	Save(user User, comment string) error

	// Delete removes a single User from the database
	Delete(user User, comment string) error

	// RequestPasswordReset handles the application-specific details of
	// delivering a password reset message to the user.
	RequestPasswordReset(user User) error

	// NewClaims generates an empty object that meets the jwt.Claims interface{}
	NewClaims() jwt.Claims

	// Close cleans up any connections opened by the service.
	Close()
}
