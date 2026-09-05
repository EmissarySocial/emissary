package mailchimp

import (
	"net/http"

	"github.com/benpate/derp"
)

// isNotFound returns TRUE if a failed request reported that the resource does not exist
func isNotFound(err error) bool {
	return derp.ErrorCode(err) == http.StatusNotFound
}

// describeError converts a failed Mailchimp request into an error that a settings form
// can show the User without rewording
func describeError(err error, location string, message string) error {

	// RULE: the four statuses below are the User's input being wrong, so each one answers
	// with an instruction. They are 422s, which puts the sentence at the ROOT of the chain
	// where `inlineErrorMessage` reads it -- a wrapper further up cannot replace it.
	switch statusCode := derp.ErrorCode(err); statusCode {

	case http.StatusUnauthorized, http.StatusForbidden:
		return derp.Validation("Mailchimp did not accept this API key. Check that you copied all of it, and that it is still listed in your Mailchimp account.", statusCode, derp.WithLocation(location))

	case http.StatusNotFound:
		return derp.Validation("Mailchimp has no account in this data center. Check the value at the start of your own Mailchimp web address, such as 'us6' in 'us6.admin.mailchimp.com'.", statusCode, derp.WithLocation(location))

	case http.StatusTooManyRequests:
		return derp.Validation("Mailchimp is rate limiting this account right now. Please wait a few minutes and try again.", statusCode, derp.WithLocation(location))
	}

	// Anything else is Mailchimp's problem or the network's, not something the User can fix
	// by editing this form, so it keeps its cause and reads as a generic failure.
	return derp.Wrap(err, location, message, derp.WithBadGateway())
}
