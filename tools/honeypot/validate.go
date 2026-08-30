// Package honeypot rejects form submissions that fill in a decoy field, which a human never
// sees and a naive bot cannot resist.
package honeypot

import (
	"net/http"
	"net/url"

	"github.com/benpate/derp"
	"github.com/benpate/re"
)

// ValidateValues returns an error if any banned field has been populated
func ValidateValues(values url.Values, bannedFields ...string) error {

	const location = "honeypot.ValidateValues"

	// Verify that banned fields are present, but NOT populated
	for _, field := range bannedFields {
		if values.Get(field) != "" {
			return derp.BadRequest(location, "Honeypot field is not empty", field)
		}
	}

	return nil
}

// Validate returns an error if any banned fields have been populated in the request body
func Validate(request *http.Request, bannedFields ...string) error {

	const location = "honeypot.Validate"

	// Read the request body (capped to guard against an oversized body)
	body, err := re.ReadRequestBody(request, re.DefaultMaximum)

	if err != nil {
		return derp.Wrap(err, location, "Reading request body")
	}

	// RULE: put back a body that can be read repeatedly.  ReadRequestBody restores a one-shot
	// reader, which the NEXT consumer drains and every one after that finds empty -- so a
	// second reader downstream would silently see an empty form.
	request.Body = re.NewReaderFromBytes(body)

	// Parse the form data
	values, err := url.ParseQuery(string(body))

	if err != nil {
		return derp.Wrap(err, location, "Unmarshalling request body")
	}

	return ValidateValues(values, bannedFields...)
}
