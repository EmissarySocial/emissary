package plugin

// PasswordRule is used to verify if a password meets the password complexity criteria for this system.
type PasswordRule interface {

	// ID returns a string that uniquely identifies this plugin.
	ID() string

	// PasswordRuleDescription returns a map of language tags to human-readable strings that explain how the password can be used
	PasswordRuleDescription(language string) string

	// ValidatePassword returns TRUE if the password can be used in this system.  If not, it returns FALSE, and a message explaining why
	ValidatePassword(password string) (OK bool, errorMessage string)
}
