package content

import (
	"context"
	"io"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/remote"
)

// HTTPS reads a single Markdown file from a public URL
type HTTPS struct {
	client          *http.Client
	allowPrivateIPs bool
}

// NewHTTPS returns an HTTPS adapter that reaches the network only through Emissary's
// SSRF-guarded HTTP client.  Plain HTTP is allowed only when allowPrivateIPs is TRUE.
func NewHTTPS(allowPrivateIPs bool) HTTPS {
	return HTTPS{
		client:          remote.NewHTTPClient(allowPrivateIPs),
		allowPrivateIPs: allowPrivateIPs,
	}
}

// Protocol returns the StreamSource Method that this adapter reads
func (adapter HTTPS) Protocol() string {
	return model.StreamSourceMethodHTTPS
}

// Version returns the validator that the origin offers for this source, or an empty string
// when it offers none.
func (adapter HTTPS) Version(ctx context.Context, source model.StreamSource) (string, error) {

	const location = "content.HTTPS.Version"

	response, err := adapter.request(ctx, source.URL, source.Version)

	if err != nil {
		return "", derp.Wrap(err, location, "Requesting source")
	}

	defer closeBody(response)

	// RULE: The origin says nothing changed, so the stored validator still stands
	if response.StatusCode == http.StatusNotModified {
		return source.Version, nil
	}

	if err := checkStatus(response, location); err != nil {
		return "", err
	}

	// An origin that offers no validator returns "", which is not an error -- it means every
	// ping fetches, and ContentHash decides whether anything actually happened.
	return response.Header.Get("ETag"), nil
}

// Fetch returns the Markdown that this source holds now.  The version argument is accepted for
// the Adapter interface and unused: an HTTPS source can only serve what it holds at this moment.
func (adapter HTTPS) Fetch(ctx context.Context, source model.StreamSource, _ string) (Item, error) {

	const location = "content.HTTPS.Fetch"

	response, err := adapter.request(ctx, source.URL, "")

	if err != nil {
		return Item{}, derp.Wrap(err, location, "Requesting source")
	}

	defer closeBody(response)

	if err := checkStatus(response, location); err != nil {
		return Item{}, err
	}

	// RULE: The format comes from the ORIGINAL address first and the declared media type second.
	// Every forge serves a raw file as text/plain whatever it holds, so only the extension
	// separates a .md from a .html.  Decided before the body is read, so a media type this
	// package cannot use is refused without downloading it.
	format, err := contentFormat(source.URL, response.Header.Get("Content-Type"))

	if err != nil {
		return Item{}, derp.Wrap(err, location, "Unusable content", source.URL)
	}

	// Read one byte past the cap, so that a body AT the limit is kept and a larger one is refused
	body, err := io.ReadAll(io.LimitReader(response.Body, maxContentBytes+1))

	if err != nil {
		return Item{}, derp.Wrap(err, location, "Reading source")
	}

	if len(body) > maxContentBytes {
		return Item{}, derp.BadRequest(location, "File is too large", source.URL, maxContentBytes)
	}

	return NewItem(format, body)
}

// FetchFile opens one file that a source links to, such as an image in its attachments folder.
// The type is not checked here: forges serve binaries as application/octet-stream, so the caller
// decides from the bytes.
func (adapter HTTPS) FetchFile(ctx context.Context, address string, maxBytes int64) (io.ReadCloser, error) {

	const location = "content.HTTPS.FetchFile"

	response, err := adapter.request(ctx, address, "")

	if err != nil {
		return nil, derp.Wrap(err, location, "Requesting file")
	}

	if err := checkStatus(response, location); err != nil {
		closeBody(response)
		return nil, err
	}

	// RULE: A declared size over the cap is refused before a byte of it is read
	if response.ContentLength > maxBytes {
		closeBody(response)
		return nil, derp.BadRequest(location, "File is too large", maxBytes)
	}

	return newCappedReader(response.Body, maxBytes), nil
}

// Subscribe returns a NotImplemented error, because a file cannot offer a subscription.
// Notification is arranged by hand through the webhook instead.
func (adapter HTTPS) Subscribe(_ context.Context, _ model.StreamSource, _ string) error {
	return derp.NotImplemented("content.HTTPS.Subscribe", "An HTTPS source cannot push notifications")
}

/******************************************
 * Helper Methods
 ******************************************/

// request issues a GET for an address, sending ifNoneMatch as a conditional header when it is
// not empty.  Every address passes the same rules, whether it names the source or a file it links to.
func (adapter HTTPS) request(ctx context.Context, sourceURL string, ifNoneMatch string) (*http.Response, error) {

	const location = "content.HTTPS.request"

	address, err := parseSourceURL(sourceURL, adapter.allowPrivateIPs)

	if err != nil {
		return nil, derp.Wrap(err, location, "Invalid source address")
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)

	if err != nil {
		return nil, derp.Wrap(err, location, "Building request")
	}

	if ifNoneMatch != "" {
		request.Header.Set("If-None-Match", ifNoneMatch)
	}

	response, err := adapter.client.Do(request)

	if err != nil {
		// A transport failure is OURS to retry, not the author's to fix, so it stays a 500
		return nil, derp.Wrap(err, location, "Unable to reach source")
	}

	// RULE: Do's signature permits a nil response beside a nil error, and every caller here
	// dereferences what it gets back.  Settling it once keeps the guard out of all of them.
	if response == nil {
		return nil, derp.Internal(location, "Source returned no response")
	}

	return response, nil
}

// Verify that HTTPS satisfies the Adapter interface
var _ Adapter = HTTPS{}
