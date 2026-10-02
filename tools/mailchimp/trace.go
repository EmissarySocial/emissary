package mailchimp

import (
	"github.com/benpate/derp"
	"github.com/rs/zerolog/log"
)

// traceRequest logs the outcome of one member request, including Mailchimp's own error body.
// TEMPORARY (Mailchimp sync diagnosis): remove once the sync is confirmed working
func traceRequest(location string, method string, url string, err error) {

	// derp.NewHTTPError has already redacted the Authorization header by the time it gets here
	log.Info().
		Str("trace", "MailchimpTrace").
		Str("step", "5-http").
		Str("location", location).
		Str("method", method).
		Str("url", url).
		Bool("failed", err != nil).
		Int("errorCode", derp.ErrorCode(err)).
		Str("error", derp.Serialize(err)).
		Msg("MailchimpTrace: Mailchimp API request finished")
}
