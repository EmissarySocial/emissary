package asnormalizer

import (
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/property"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

type Client struct {
	innerClient streams.Client
}

func New(innerClient streams.Client) *Client {
	return &Client{
		innerClient: innerClient,
	}
}

func (client *Client) Load(uri string, options ...any) (streams.Document, error) {

	const location = "asnormalizer.Client.Load"

	// Forward request to inner client
	result, err := client.innerClient.Load(uri, options...)

	if err != nil {
		return streams.NilDocument(), derp.Wrap(err, location, "Error loading document from inner client", uri)
	}

	// Try to Normalize the document
	if normalized := Normalize(result); normalized != nil {
		result.SetValue(property.Map(normalized))
	}

	// Return the result
	return result, nil
}

func Normalize(document streams.Document) map[string]any {

	switch {

	// All Actor types (Person, Organization, Application, etc)
	case document.IsActor():
		return Actor(document)

	// Regular documents here (Article, Note, etc)
	case document.IsObject():
		return Object(document)

		// TODO: Normalize Collections?
	}

	switch document.Type() {

	// Likes (treat EmojiReactions as likes)
	case vocab.ActivityTypeLike,
		"EmojiReact",
		"EmojiReaction":
		return Like(document)

	// Dislikes
	case vocab.ActivityTypeDislike:
		return Dislike(document)

	// Creates/Updates are treated like an Object.  This may be
	// skipped by the Object() function if the document does not match
	case vocab.ActivityTypeCreate,
		vocab.ActivityTypeUpdate:

		return Object(document)
	}

	// Unrecognized documents return nil, which will be ignored by the caller
	return nil
}
