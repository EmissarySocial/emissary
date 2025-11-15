package honeypot

import (
	"net/http"
	"net/url"

	"github.com/benpate/derp"
	"github.com/benpate/re"
)

// Validate returns an error if any banned fields have been populated
func Validate(request *http.Request, bannedFields ...string) error {

	const location = "honeypot.Prevent"

	// Read the request body
	body, err := re.ReadRequestBody(request)

	if err != nil {
		return derp.Wrap(err, location, "Unable to read request body")
	}

	// Parse the form data
	values, err := url.ParseQuery(string(body))

	if err != nil {
		return derp.Wrap(err, location, "Error unmarshalling request body")
	}

	// Verify that banned fields are present, but NOT populated
	for _, field := range bannedFields {

		if values.Get(field) != "" {
			return derp.BadRequestError(location, "Honeypot field is not empty", field, values.Get(field))
		}
	}

	return nil
}
