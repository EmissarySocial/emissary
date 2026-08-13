// Package ascacherules is a streams.Client middleware that rewrites the Cache-Control header of every
// document passing through it, imposing Emissary's own caching policy on top of whatever the origin
// asked for.  It sits directly above the cache itself, so the windows decided here are the windows
// the cache stores by.
package ascacherules

import (
	"github.com/EmissarySocial/emissary/tools/cacheheader"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// Client is a streams.Client wrapper that applies Emissary's caching rules to loaded documents.
type Client struct {
	innerClient streams.Client
}

// New creates a fully initialized Client object.
func New(innerClient streams.Client) *Client {
	result := &Client{
		innerClient: innerClient,
	}

	result.innerClient.SetRootClient(result)
	return result
}

// SetRootClient applies a "top level" client to the rest of the client stack.
func (client *Client) SetRootClient(rootClient streams.Client) {
	if client.innerClient != nil {
		client.innerClient.SetRootClient(rootClient)
	}
}

// Load retrieves a document from the inner client, then rewrites its Cache-Control header to match
// the rules for that document's type.
func (client *Client) Load(uri string, options ...any) (streams.Document, error) {

	// Retrieve the actual document from the inner client.
	result, err := client.innerClient.Load(uri, options...)

	if err != nil {
		return result, err
	}

	header := result.HTTPHeader()
	cacheControl := cacheheader.Parse(header)

	switch {

	// Activity objects are never cached.  This prevents likes,
	// reposts, and other actions from being cached.
	case result.IsActivity():

		cacheControl.MaxAge = 0
		cacheControl.NoStore = true

	// Collections (et al) are cached for up to one minute. This
	// minimizes traffic on heavy loads, but keeps collections
	// refreshed in (near) real-time
	case result.IsCollection():

		cacheControl.MaxAge = clamp(0, cacheControl.MaxAge, minute)

	// Actors are complicated.  IF they include an outbox (a la sherlock/RSS feeds)
	// then we cache them like they ARE a collection.  Otherwise, it's Okay (I think)
	// to cache it for longer
	case result.IsActor():

		// This happens when we're "faking" an actor via RSS.  These carry no keys, so their
		// staleness is a display concern only, and the old window still applies.
		if result.Outbox().IsMap() {
			cacheControl.MaxAge = clamp(hour, cacheControl.MaxAge, day)
			break
		}

		// This is a normal ActivityPub lookup
		cacheControl.MaxAge = actorMaxAge(cacheControl)

	// Tombstone documents are cached for at least one month, and up to one year.
	case result.Type() == vocab.ObjectTypeTombstone:
		cacheControl.MaxAge = clamp(month, cacheControl.MaxAge, year)

	// All other items (Articles, Notes, etc) are cached for at least one day
	// and up to one month
	default:
		cacheControl.MaxAge = clamp(day, cacheControl.MaxAge, month)
	}

	// Write the cacheControl value back into the document header
	header.Set("Cache-Control", cacheControl.String())
	result.SetHTTPHeader(header)

	// Return the result
	return result, nil
}

// actorMaxAge returns the number of seconds that an ActivityPub Actor document may be cached.
func actorMaxAge(cacheControl cacheheader.Header) int64 {

	// This ceiling is a SECURITY bound, not a performance knob: the Actor document carries the public
	// key that authenticates everything the Actor sends, so its cache window is also the window in
	// which a revoked key keeps being accepted. The old rule promoted a peer that asked for
	// `max-age=180` to a full day, and allowed up to a month -- overriding an origin that was
	// actively asking to stay fresh. (BUG-22 D2)

	// A shared cache prefers s-maxage.  An absent header parses as zero, indistinguishable from an
	// explicit zero, so silence means "use our default" rather than "do not cache".
	stated := cacheControl.SMaxAge

	if stated == 0 {
		stated = cacheControl.MaxAge
	}

	if stated == 0 {
		return 12 * hour
	}

	// An origin that states its own freshness is honored, within one minute and one day.
	return clamp(minute, stated, day)
}

// Save forwards to the inner client.  This middleware rewrites reads, not writes.
func (client *Client) Save(document streams.Document) error {
	return client.innerClient.Save(document)
}

// Delete forwards to the inner client.
func (client *Client) Delete(documentID string) error {
	return client.innerClient.Delete(documentID)
}
