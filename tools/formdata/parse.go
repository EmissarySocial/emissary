// Package formdata parses an HTML form submission from an HTTP request, in either of the
// encodings a browser uses to send one.
package formdata

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/benpate/derp"
)

// multipartMaxMemory bounds how much of a multipart form is buffered in memory before
// the remainder spills to temporary files
const multipartMaxMemory = 8 << 20 // 8MB

// Parse returns the values from a form submission, MERGING the URL query string into the
// values sent in the request body.  Callers that read a control parameter from the query
// string depend on this merge.
func Parse(request *http.Request) (url.Values, error) {

	if err := parse(request, "formdata.Parse"); err != nil {
		return url.Values{}, err
	}

	return request.Form, nil
}

// ParseBody returns ONLY the values sent in the request body, ignoring the URL query string.
// Use this for values supplied by an untrusted visitor: because Parse merges the two and a
// repeated name yields BOTH values, a crafted link can otherwise append its own text to a
// field that the person filling out the form believes only they control.
func ParseBody(request *http.Request) (url.Values, error) {

	if err := parse(request, "formdata.ParseBody"); err != nil {
		return url.Values{}, err
	}

	// RULE: ParseForm and ParseMultipartForm both populate PostForm, but a request that
	// carries no body at all leaves it nil.  Return an empty set so callers can range
	// over the result without a nil check.
	if request.PostForm == nil {
		return url.Values{}, nil
	}

	return request.PostForm, nil
}

// parse decodes the request body into the Request's cached Form and PostForm values.
// Both are cached on the Request, so calling this more than once is safe and re-reads nothing.
func parse(request *http.Request, location string) error {

	contentType := request.Header.Get("Content-Type")

	// Try to parse URL encoded Values
	if contentType == "application/x-www-form-urlencoded" {

		if err := request.ParseForm(); err != nil {
			return derp.Wrap(err, location, "Parsing form body")
		}

		return nil
	}

	// Try to parse multipart form data
	if strings.HasPrefix(contentType, "multipart/form-data") {

		if err := request.ParseMultipartForm(multipartMaxMemory); err != nil {
			return derp.Wrap(err, location, "Parsing multipart form")
		}

		return nil
	}

	// Unrecognized content type
	return derp.BadRequest(location, "Unsupported encoding", contentType)
}
