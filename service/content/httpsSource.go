package content

import (
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
)

// parseSourceURL validates a source address and returns it in normalized form
func parseSourceURL(value string, allowPrivateIPs bool) (string, error) {

	const location = "content.parseSourceURL"

	if len(value) > maxURLLength {
		return "", derp.BadRequest(location, "Address is too long", maxURLLength)
	}

	// The parser's own error quotes the whole address, which can carry a password, so it is
	// replaced rather than wrapped
	parsed, err := url.Parse(value)

	if err != nil {
		return "", derp.BadRequest(location, "Address is not a valid URL")
	}

	// RULE: Plain HTTP is allowed only where private addresses are, which is local development
	switch parsed.Scheme {

	case "https":

	case "http":
		if !allowPrivateIPs {
			return "", derp.BadRequest(location, "Address must use https")
		}

	default:
		return "", derp.BadRequest(location, "Address must use https", parsed.Scheme)
	}

	if parsed.Host == "" {
		return "", derp.BadRequest(location, "Address must name a host")
	}

	// RULE: Credentials never live in the address, where they would be stored in plain text
	if parsed.User != nil {
		return "", derp.BadRequest(location, "Address must not include a username or password")
	}

	return parsed.String(), nil
}

// contentFormat maps the media type a server declared onto a model ContentFormat, and refuses
// anything this version cannot use
func contentFormat(header string) (string, error) {

	const location = "content.contentFormat"

	// RULE: A server that declares nothing is refused rather than guessed at.  Sniffing the bytes
	// would accept the HTML page that C2 exists to catch, because it starts with plain text.
	if header == "" {
		return "", derp.BadRequest(location, "Source did not declare a Content-Type")
	}

	mediaType, _, err := mime.ParseMediaType(header)

	if err != nil {
		return "", derp.BadRequest(location, "Source declared an unreadable Content-Type", header)
	}

	switch strings.ToLower(mediaType) {

	case mediaTypeMarkdown, "text/x-markdown":
		return model.ContentFormatMarkdown, nil

	// Every forge surveyed serves a raw .md file as text/plain, so this is the common case
	case mediaTypePlain:
		return model.ContentFormatMarkdown, nil
	}

	return "", derp.BadRequest(location, "Source is not Markdown", mediaType)
}

// checkStatus converts a non-200 response into an error that says whose problem it is
func checkStatus(response *http.Response, location string) error {

	if response.StatusCode == http.StatusOK {
		return nil
	}

	// RULE: A 5xx is Emissary's to retry, so it stays a 500 and is filed as a defect.  Everything
	// else is the author's address to fix, and is shown to them as a status message.
	if response.StatusCode >= 500 {
		return derp.Internal(location, "Source server failed", response.Status)
	}

	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return derp.BadRequest(location, "Source is not public", response.Status)
	}

	if response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusGone {
		return derp.NotFound(location, "Source not found", response.Status)
	}

	return derp.BadRequest(location, "Source could not be read", response.Status)
}

// closeBody drains and closes a response body, so the connection can be reused
func closeBody(response *http.Response) {

	if response.Body == nil {
		return
	}

	// Drain before closing, so the connection can be reused
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maxContentBytes))
	_ = response.Body.Close()
}
