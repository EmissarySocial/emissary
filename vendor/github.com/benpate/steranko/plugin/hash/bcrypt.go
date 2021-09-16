package hash

import (
	"github.com/benpate/derp"
	"golang.org/x/crypto/bcrypt"
)

// BCrypt is the default password encryption scheme for Steranko.  The integer value represents the
// complexity cost of the algorithm.
type BCrypt int

// ID returns a unique identifier for this plugin.
func (bc BCrypt) ID() string {
	return "BCrypt"
}

// HashPassword returns a hashed value for the password.
func (bc BCrypt) HashPassword(plaintext string) (hashedValue string, error error) {

	result, err := bcrypt.GenerateFromPassword([]byte(plaintext), int(bc))

	if err != nil {
		return "", derp.New(500, "steranko.plugin.hash.HashPassword", "Error hashing plaintext", err)
	}

	return string(result), nil
}

// CompareHashedPassword checks that a hashedValue value matches the plaintext password.
func (bc BCrypt) CompareHashedPassword(hashedValue string, plaintext string) (OK bool, rehash bool) {

	// Try to validate the password.  If it cannot be matched, then return failure.
	if err := bcrypt.CompareHashAndPassword([]byte(hashedValue), []byte(plaintext)); err != nil {

		// FALSE, FALSE means that the password is not OK.
		return false, false
	}

	// Try to compute the password cost.
	if cost, err := bcrypt.Cost([]byte(hashedValue)); cost < int(bc) {

		// Silently report this error because we don't want to interrupt the application flow.
		derp.Report(derp.New(500, "steranko.plugin.hash.CompareHashedPassword", "Error generating password cost", err))

		// TRUE, TRUE means that the password is OK, but needs to be re-hashed
		return true, true
	}

	// TRUE, FALSE means that the password is OK, and doesn't need to be re-hashed
	return true, false
}
