// Package asblock provides a hannibal streams.Client middleware that refuses to fetch documents from
// blocked origins (R19). It is inserted BELOW the cache in the client stack, so cache hits are served
// without consulting it -- only live network fetches are gated, and a blocked origin is never contacted.
package asblock

import (
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
)

// BlockChecker reports whether a URI's origin is blocked and must not be fetched. It is supplied by the
// service layer, which holds the rules and a database session, so this package needs no knowledge of
// how blocks are stored or which User the fetch is for.
type BlockChecker func(uri string) (bool, error)

// Client is the streams.Client middleware. It gates Load (network fetches) and passes Save/Delete
// through unchanged.
type Client struct {
	innerClient streams.Client
	isBlocked   BlockChecker
}

// New returns a Client that gates the innerClient's fetches with the provided BlockChecker.
func New(innerClient streams.Client, isBlocked BlockChecker) *Client {
	result := &Client{
		innerClient: innerClient,
		isBlocked:   isBlocked,
	}

	result.innerClient.SetRootClient(result)
	return result
}

// SetRootClient forwards the top-level client pointer to the inner client.
func (client *Client) SetRootClient(rootClient streams.Client) {
	if client.innerClient != nil {
		client.innerClient.SetRootClient(rootClient)
	}
}

// Load refuses the fetch if the URI's origin is blocked; otherwise it delegates to the inner client.
func (client *Client) Load(uri string, options ...any) (streams.Document, error) {

	const location = "asblock.Client.Load"

	blocked, err := client.isBlocked(uri)

	// RULE: fail OPEN. A rule-check failure must not halt all outbound federation; the primary
	// protection is upstream (the ingest walk's own pre-fetch checks), so this backstop reports the
	// error and allows the fetch rather than breaking every fetch on a database blip.
	if err != nil {
		derp.Report(derp.Wrap(err, location, "Error checking block rules before fetch; allowing", uri))
		return client.innerClient.Load(uri, options...)
	}

	// RULE: never contact a blocked origin (R19).
	if blocked {
		return streams.NilDocument(), derp.Forbidden(location, "Refusing to fetch document from a blocked origin", uri)
	}

	return client.innerClient.Load(uri, options...)
}

// Save forwards to the inner client (blocking gates reads, not cache writes).
func (client *Client) Save(document streams.Document) error {
	return client.innerClient.Save(document)
}

// Delete forwards to the inner client.
func (client *Client) Delete(documentID string) error {
	return client.innerClient.Delete(documentID)
}
