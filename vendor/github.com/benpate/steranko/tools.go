package steranko

import (
	"github.com/benpate/derp"
	"github.com/benpate/schema"
)

// Authenticate verifies a username/password combination.
func (s *Steranko) Authenticate(username string, password string, user User) error {

	// Try to load the User from the UserService
	if err := s.UserService.Load(username, user); err != nil {

		if derp.NotFound(err) {
			return derp.New(CodeUnauthorized, "steranko.Authenticate", "Unauthorized", username, "user not found")
		}

		return derp.Wrap(err, "steranko.Authenticate", "Error loading User account", username, "database error")
	}

	// Fall through means that we have a matching user account.

	// Try to authenticate the password
	ok, update := s.PasswordHasher.CompareHashedPassword(password, user.GetPassword())

	if !ok {
		return derp.New(CodeUnauthorized, "steranko.Authenticate", "Unauthorized", username, "invalid password")
	}

	if update {

		if hashedValue, err := s.PasswordHasher.HashPassword(password); err == nil {
			user.SetPassword(hashedValue)
			_ = s.UserService.Save(user, "Password automatically upgraded by Steranko")
			// Intentionally ignoring errors updating the password because the user has already
			// authenticated.  If we can't update it now (for some reason) then we'll get it soon.
		}
	}

	return nil
}

// ValidatePassword checks a password against the requirements in the Config structure.
func (s *Steranko) ValidatePassword(password string) error {

	if err := s.PasswordSchema().Validate(password); err != nil {
		return derp.Wrap(err, "steranko.ValidatePassword", "Password does not meet requirements")
	}

	return nil
}

// PasswordSchema returns the schema.Schema for validating passwords
func (s *Steranko) PasswordSchema() *schema.Schema {

	if s.passwordSchema == nil {
		s.passwordSchema = &s.Config.PasswordSchema
	}

	return s.passwordSchema
}
