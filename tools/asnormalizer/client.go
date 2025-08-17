package asnormalizer

import (
	"strconv"

	"github.com/benpate/derp"
	"github.com/benpate/hannibal/property"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/cespare/xxhash/v2"
	"github.com/davecgh/go-spew/spew"
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

	// original := result.Clone().Map()

	// Try to Normalize the document
	if normalized := Normalize(client.rootClient, result); normalized != nil {
		result.SetValue(property.Map(normalized))
	}

	// NOW LETS CALCULATE SOME METADATA (OBJECTS ONLY)
	if result.IsObject() {

		// Calculate the HashedID
		hashedID := xxhash.Sum64String(result.ID())
		hashedIDString := strconv.FormatUint(hashedID, 32)
		result.Metadata.HashedID = hashedIDString

		// Calculate the Document Category
		documentCategory := result.Type()
		result.Metadata.DocumentCategory = streams.DocumentCategory(documentCategory)

		// Calculate Relationships
		relationType, relationHref := calcRelationType(result)
		result.Metadata.RelationType = relationType
		result.Metadata.RelationHref = relationHref

		spew.Dump(location, result.ID(), result.Metadata)
	}

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

// calcRelationType calculates the "RelationType" and "RelationHref" metadata for this
// cached document.
func calcRelationType(document streams.Document) (string, string) {

	// Get the document type
	documentType := document.Type()

	// Calculate RelationType
	switch documentType {

	// Announce, Like, and Dislike are written straight to the cache.
	case vocab.ActivityTypeAnnounce,
		vocab.ActivityTypeLike,
		vocab.ActivityTypeDislike:

		return documentType, document.Object().ID()

	// Otherwise, see if this is a "Reply"
	default:
		unwrapped := document.UnwrapActivity()

		if inReplyTo := unwrapped.InReplyTo(); inReplyTo.NotNil() {
			return vocab.RelationTypeReply, inReplyTo.String()
		}
	}

	return "", ""
}
