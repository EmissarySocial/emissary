// Package asrules provides a hannibal streams.Client middleware that evaluates every document
// against the current viewer's moderation Rules. It refuses to fetch documents the viewer's rules
// hide (R19), and stamps the viewer's verdict -- a metadata.LabelSet of hide-reasons and
// annotations -- into each returned document's Metadata, where later layers can act on it. It sits
// ABOVE the cache in the client stack, so cache hits and network fetches are annotated alike, and
// per-viewer results never touch the shared cache.
package asrules

import (
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/metadata"
	"github.com/benpate/hannibal/streams"
)

// Checker computes the rule verdict for a document. Before the fetch it is called with a
// NilDocument (only the URL is known); after a successful load it is called again with the full
// document, whose author and tags can match additional rules. It is supplied by the service layer,
// which holds the rules and a database session, so this package needs no knowledge of how rules are
// stored or which viewer the fetch is for.
type Checker func(uri string, document streams.Document) (metadata.LabelSet, error)

// Client is the streams.Client middleware. It gates and annotates Load, and passes Save/Delete
// through unchanged.
type Client struct {
	innerClient streams.Client
	checker     Checker
}

// New returns a Client that wraps the innerClient with the provided rule Checker.
func New(innerClient streams.Client, checker Checker) *Client {
	result := &Client{
		innerClient: innerClient,
		checker:     checker,
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

// Load evaluates the URI against the viewer's rules before fetching (a hidden verdict refuses the
// fetch), then re-evaluates the loaded document and stamps the verdict into its Metadata.
func (client *Client) Load(uri string, options ...any) (streams.Document, error) {

	const location = "asrules.Client.Load"

	// Evaluate the URL's own keys before anything is fetched.
	verdict, err := client.checker(uri, streams.NilDocument())

	// RULE: fail OPEN. A rule-check failure must not halt all outbound federation; the primary
	// protection is upstream (the ingest walk's own pre-fetch checks), so this backstop reports the
	// error and proceeds rather than breaking every fetch on a database blip.
	if err != nil {
		derp.Report(derp.Wrap(err, location, "Error checking rules before fetch; proceeding", uri))
		verdict = nil
	}

	// RULE: never fetch a document the viewer's rules hide (R19) -- unless the caller reveals it,
	// which is the render layer's click-to-reveal path (D2). The refused document still carries the
	// verdict, so the UX can render an attributed placeholder.
	if verdict.IsHidden() {
		if config := newLoadConfig(options...); !config.reveal {
			document := streams.NilDocument()
			document.Metadata.Labels = verdict
			return document, derp.Forbidden(location, "Refusing to fetch a document hidden by the viewer's rules", uri)
		}
	}

	// Pass the request down the chain (inner errors pass through unchanged).
	document, err := client.innerClient.Load(uri, options...)

	if err != nil {
		return document, err
	}

	// Re-evaluate with the loaded document's own keys (author, tags). A failure here also fails
	// open: keep the URL-level verdict already in hand.
	if refined, err := client.checker(uri, document); err == nil {
		verdict = refined
	} else {
		derp.Report(derp.Wrap(err, location, "Error checking rules for loaded document; using URL-level verdict", uri))
	}

	// Stamp the verdict into the document's per-viewer Metadata.
	document.Metadata.Labels = verdict

	// The Dude abides.
	return document, nil
}

// Save forwards to the inner client (rules gate reads, not cache writes).
func (client *Client) Save(document streams.Document) error {
	return client.innerClient.Save(document)
}

// Delete forwards to the inner client.
func (client *Client) Delete(documentID string) error {
	return client.innerClient.Delete(documentID)
}
