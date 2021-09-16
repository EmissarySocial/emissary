package steranko

import (
	"github.com/benpate/schema"
	"github.com/benpate/steranko/plugin"
	"github.com/benpate/steranko/plugin/hash"
)

// Steranko contains all required configuration information for this library.
type Steranko struct {
	UserService    UserService           // Service that provides CRUD operations on Users
	KeyService     KeyService            // Service that generates/retrieves encryption keys used in JWT signatures.
	Config         Config                // Configuration options for this library
	PasswordHasher plugin.PasswordHasher // PasswordHasher uses a one-way encryption to obscure stored passwords.
	PasswordRules  []plugin.PasswordRule // PasswordRules provide rules for enforcing password complexity

	passwordSchema *schema.Schema
}

// New returns a fully initialized Steranko instance, with HandlerFuncs that support all of your user authentication and authorization needs.
func New(userService UserService, keyService KeyService, config Config) *Steranko {

	result := Steranko{
		UserService: userService,
		KeyService:  keyService,
		Config:      config,

		// PasswordHasher: hash.BCrypt(15),
		PasswordHasher: hash.Plaintext{},
		PasswordRules:  []plugin.PasswordRule{},
	}

	// Parse password rules from config file here

	return &result
}
