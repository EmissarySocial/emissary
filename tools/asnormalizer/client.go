package asnormalizer

import (
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/property"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// Client is a streams.Client decorator that normalizes every document it loads into a predictable shape
type Client struct {
	rootClient  streams.Client
	innerClient streams.Client
}

// New returns a fully initialized normalizer Client that wraps the provided innerClient
func New(innerClient streams.Client) *Client {
	result := &Client{
		innerClient: innerClient,
	}

	result.innerClient.SetRootClient(result)
	return result
}

// SetRootClient implements the streams.Client interface, and passes the root client down the chain
func (client *Client) SetRootClient(rootClient streams.Client) {
	client.rootClient = rootClient
	if client.innerClient != nil {
		client.innerClient.SetRootClient(rootClient)
	}
}

// Load implements the streams.Client interface, normalizing the document and populating its Metadata
func (client *Client) Load(uri string, options ...any) (streams.Document, error) {

	const location = "asnormalizer.Client.Load"

	defer func() {
		if err := recover(); err != nil {
			derp.Report(derp.Internal(location, "Recovered error", err, uri))
		}
	}()

	// Collect the documents already being loaded above this one
	config := newLoadConfig(options...)

	// RULE: A document already loading above this one gets a stub. Loading it again would wait
	// in the Carpool on its own leader, or recurse forever without one.
	if config.contains(uri) {
		return stub(uri), nil
	}

	// Forward request to inner client
	result, err := client.innerClient.Load(uri, options...)

	if err != nil {
		return streams.NilDocument(), derp.Wrap(err, location, "Loading document from inner client", uri)
	}

	// Work on a copy bound to a client that passes the history into every load it starts, including
	// hidden loads from getters on bare-URL values. The id catches Create unwrapping and redirects.
	originalClient := result.Client()
	config.remember(uri, result.ID())
	historyClient := streams.NewOptionsClient(client.rootClient, WithHistory(config.history...))
	result = result.AddOptions(streams.WithClient(historyClient))

	// Try to Normalize the document
	if normalized := Normalize(historyClient, result); normalized != nil {
		result.SetValue(property.Map(normalized))
	}

	// Write the metadata that is derived from the document itself
	Stamp(&result)

	// Return the result bound to its original client, so that nothing downstream inherits the history
	return result.AddOptions(streams.WithClient(originalClient)), nil
}

// Save implements the streams.Client interface, and passes the document to the innerClient
func (client *Client) Save(document streams.Document) error {
	return client.innerClient.Save(document)
}

// Delete implements the streams.Client interface, and passes the documentID to the innerClient
func (client *Client) Delete(documentID string) error {
	return client.innerClient.Delete(documentID)
}

// Normalize converts a document into a plain map, using the normalizer that matches the document type
func Normalize(rootClient streams.Client, document streams.Document) map[string]any {

	switch {

	// All Actor types (Person, Organization, Application, etc)
	case document.IsActor():
		return Actor(document)

	// Regular documents here (Article, Note, etc)
	case document.IsObject():
		return Object(rootClient, document)

	// Collections (OrderedCollection, Collection, etc)
	case document.IsCollection():
		// TODO:
	}

	switch document.Type() {

	// Likes (treat EmojiReactions as likes)
	case vocab.ActivityTypeLike,
		vocab.ActivityTypeEmojiReact,
		vocab.ActivityTypeEmojiReactAlt:

		return Like(document)

	// Dislikes
	case vocab.ActivityTypeDislike:
		return Dislike(document)

	// Creates/Updates are treated like an Object.  This may be
	// skipped by the Object() function if the document does not match
	case vocab.ActivityTypeCreate,
		vocab.ActivityTypeUpdate:

		return Object(rootClient, document)
	}

	// Unrecognized documents return nil, which will be ignored by the caller
	return nil
}
