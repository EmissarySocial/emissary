package asnormalizer

import (
	"strconv"

	"github.com/benpate/derp"
	"github.com/benpate/hannibal/property"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/cespare/xxhash/v2"
)

type Client struct {
	rootClient  streams.Client
	innerClient streams.Client
}

func New(innerClient streams.Client) *Client {
	result := &Client{
		innerClient: innerClient,
	}

	result.innerClient.SetRootClient(result)
	return result
}

func (client *Client) SetRootClient(rootClient streams.Client) {
	client.rootClient = rootClient
	client.innerClient.SetRootClient(rootClient)
}

func (client *Client) Load(uri string, options ...any) (streams.Document, error) {

	const location = "asnormalizer.Client.Load"

	// Forward request to inner client
	result, err := client.innerClient.Load(uri, options...)

	if err != nil {
		return streams.NilDocument(), derp.Wrap(err, location, "Error loading document from inner client", uri)
	}

	original := result.Clone().Map()

	// Try to Normalize the document
	if normalized := Normalize(client.rootClient, result); normalized != nil {
		result.SetValue(property.Map(normalized))
	}

	// Add a hashed representation of the ID for (easier?) lookups?
	hashedID := xxhash.Sum64String(result.ID())
	hashedIDString := strconv.FormatUint(hashedID, 32)
	result.SetString("x-hashed-id", hashedIDString)
	result.SetProperty("x-original", original)

	// Return the result
	return result, nil
}

func Normalize(client streams.Client, document streams.Document) map[string]any {

	switch {

	// All Actor types (Person, Organization, Application, etc)
	case document.IsActor():
		return Actor(document)

	// Regular documents here (Article, Note, etc)
	case document.IsObject():
		return Object(client, document)

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

		return Object(client, document)
	}

	// Unrecognized documents return nil, which will be ignored by the caller
	return nil
}
