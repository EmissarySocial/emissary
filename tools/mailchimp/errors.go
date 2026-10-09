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

// describeMemberError converts a failed member request into an error that keeps Mailchimp's
// own status code, so the queue can tell a retry from a permanent failure
func describeMemberError(err error, location string, message string) error {

	statusCode := derp.ErrorCode(err)

	// RULE: a transport failure carries no status of its own, and is always worth retrying
	if statusCode < 400 {
		return derp.BadGateway(location, message)
	}

	// RULE: the status code is the whole point here, and `describeError` would erase it.
	// `requeue` reads it to tell a rate limit (retry) from a bad request (give up), and
	// `mailchimp_reportMemberError` reads 401/403 to tell a revoked key from a blip.
	// The failed transaction is deliberately left out: it carries an Authorization header,
	// and nobody who reads this path would benefit from the detail.
	return derp.Internal(location, message, derp.WithCode(statusCode))
}
