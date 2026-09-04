package mailchimp

import (
	"strings"

	"github.com/benpate/derp"
)

// apiKeyMaxLength bounds a credential so an absurd paste cannot become an absurd
// HTTP header
const apiKeyMaxLength = 256

// ValidateAPIKey returns an error if the provided credential cannot be safely
// stored and transmitted
func ValidateAPIKey(apiKey string) error {

	const location = "tools.mailchimp.ValidateAPIKey"

	// Trim the whitespace a paste from Mailchimp's UI tends to carry.
	// Callers store the trimmed form.
	apiKey = strings.TrimSpace(apiKey)

	// RULE: a credential is required
	if apiKey == "" {
		return derp.BadRequest(location, "Mailchimp API key is required")
	}

	// RULE: bound the length before anything else reads it
	if len(apiKey) > apiKeyMaxLength {
		return derp.BadRequest(location, "Mailchimp API key is too long")
	}

	// RULE: printable ASCII only, because this travels in an Authorization header.
	// Nothing here checks FORMAT -- a credential is opaque; see README.md.
	for _, character := range apiKey {
		if character < '!' || character > '~' {
			return derp.BadRequest(location, "Mailchimp API key contains a character that cannot be sent in a request")
		}
	}

	// Shaped like something we can send. Mailchimp gets the final word.
	return nil
}
