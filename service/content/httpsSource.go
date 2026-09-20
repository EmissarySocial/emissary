package content

import (
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
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

// contentFormat decides whether a source holds Markdown or HTML, and refuses a media type that
// this version cannot read at all.
func contentFormat(sourceURL string, header string) (string, error) {

	const location = "content.contentFormat"

	// RULE: A server that declares nothing is refused rather than guessed at.  The bytes are never
	// sniffed: a Markdown file and an HTML page both begin with plain text.
	if header == "" {
		return "", derp.BadRequest(location, "Source did not declare a Content-Type")
	}

	mediaType, _, err := mime.ParseMediaType(header)

	if err != nil {
		return "", derp.BadRequest(location, "Source declared an unreadable Content-Type", header)
	}

	mediaType = strings.ToLower(mediaType)

	// RULE: The allowlist stays, so a mistyped address pointing at an image or an archive is
	// refused rather than stored as somebody's article.
	switch mediaType {

	case mediaTypeMarkdown, mediaTypeXMarkdown, mediaTypePlain, mediaTypeHTML, mediaTypeXHTML:

	default:
		return "", derp.BadRequest(location, "Source is not a text document", mediaType)
	}

	extension := sourceExtension(sourceURL)

	// RULE: A Markdown extension is absolute, and outranks the declared type.  Every forge serves
	// a raw file as text/plain whatever it holds, so the header cannot separate a .md from a
	// .html -- and the extension is what the author typed, which no forge can change.
	switch extension {

	case extensionMarkdown, extensionMarkdownLong:
		return model.ContentFormatMarkdown, nil
	}

	switch mediaType {

	case mediaTypeHTML, mediaTypeXHTML:
		return model.ContentFormatHTML, nil
	}

	switch extension {

	case extensionHTML, extensionHTMLShort:
		return model.ContentFormatHTML, nil
	}

	// Everything left is text/plain with no opinion attached, which is Markdown by default
	return model.ContentFormatMarkdown, nil
}

// sourceExtension returns the lowercased file extension of an address's path
func sourceExtension(sourceURL string) string {

	parsed, err := url.Parse(sourceURL)

	// RULE: An unparseable address is extensionless here, not an error.  parseSourceURL has
	// already refused it before any request was made, so this cannot be reached with one.
	if err != nil {
		return ""
	}

	// RULE: Read the PATH, never the whole address.  cgit serves "/plain/README.md?h=master", and
	// the text after the last dot there is "md?h=master", which matches no extension at all.
	return strings.ToLower(path.Ext(parsed.Path))
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
