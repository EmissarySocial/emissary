package asnormalizer

import (
	"strconv"

	"github.com/benpate/derp"
	"github.com/benpate/hannibal/property"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/cespare/xxhash/v2"
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

	// Forward request to inner client
	result, err := client.innerClient.Load(uri, options...)

	if err != nil {
		return streams.NilDocument(), derp.Wrap(err, location, "Loading document from inner client", uri)
	}

	// Try to Normalize the document
	if normalized := Normalize(client.rootClient, result); normalized != nil {
		result.SetValue(property.Map(normalized))
	}

	// Calculate the Document Category
	documentCategory := result.Type()
	result.Metadata.DocumentCategory = streams.DocumentCategory(documentCategory)

	// Calculate the HashedID
	hashedID := xxhash.Sum64String(result.ID())
	hashedIDString := strconv.FormatUint(hashedID, 32)
	result.Metadata.HashedID = hashedIDString

	// Additional Metadata for Objects only
	if result.IsObject() {

		// Calculate Relationships. Response COUNTS are not computed here: ascache's
		// CalcParentRelationships maintains them from locally-cached evidence, and it is the
		// single writer on purpose (two writers with different sources would fight).
		relationType, relationHref := calcRelationType(result)
		result.Metadata.RelationType = relationType
		result.Metadata.RelationHref = relationHref
	}

	// Return the result
	return result, nil
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

// calcRelationType calculates the "RelationType" and "RelationHref" metadata for this
// cached document.
func calcRelationType(document streams.Document) (string, string) {

	// Get the document type

	// Calculate RelationType
	switch documentType := document.Type(); documentType {

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
