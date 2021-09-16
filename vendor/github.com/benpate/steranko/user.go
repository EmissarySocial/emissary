package steranko

import "github.com/golang-jwt/jwt/v4"

// User interface wraps all of the functions that Steranko needs to authorize a user of the system.
// This is done so that Steranko can be retrofitted on to your existing data objects.  Just implement
// this interface, and a CRUD service, and you're all set.
type User interface {
	GetUsername() string // Returns the username of the User
	GetPassword() string // Returns the password of the User

	SetUsername(username string)   // Sets the username of the User
	SetPassword(ciphertext string) // Sets the password of the User
	Claims() jwt.Claims            // Returns all claims (permissions) that this user has.
}
