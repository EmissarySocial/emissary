package formdata

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/benpate/derp"
)

// Parse retrieves basic form values from a request body (supports)
func Parse(request *http.Request) (url.Values, error) {

	const location = "formdata.Parse"

	contentType := request.Header.Get("Content-Type")

	// Try to parse URL encoded Values
	if contentType == "application/x-www-form-urlencoded" {
		if err := request.ParseForm(); err != nil {
			return url.Values{}, derp.Wrap(err, location, "Error parsing form body")
		}

		return request.Form, nil
	}

	// Try to parse multipart form data
	if strings.HasPrefix(contentType, "multipart/form-data") {

		if err := request.ParseMultipartForm(8 << 20); err != nil {
			return url.Values{}, derp.Wrap(err, location, "Error parsing multipart form")
		}

		return request.Form, nil
	}

	// Unrecognized content type
	return url.Values{}, derp.BadRequest(location, "Unsupported encoding", contentType)
}
