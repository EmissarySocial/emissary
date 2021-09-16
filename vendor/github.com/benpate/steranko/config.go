package steranko

import "github.com/benpate/schema"

type Config struct {
	Token          string        `json:"token"`          // Where to store authentication tokens.  Valid values are HEADER (default value) or COOKIE
	PasswordSchema schema.Schema `json:"passwordSchema"` // JSON-encoded schema for password validation rules.
}
