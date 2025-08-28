package ascontextmaker

import (
	"github.com/benpate/data"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// asContextMaker is a hannibal.Streams middleware that adds a "context" property to all documents
// based on their "InReplyTo" property.  If a document does not have a context or inReplyTo, then
// it is its own context, and is updated to reflect that.
type Client struct {
	rootClient     streams.Client
	innerClient    streams.Client
	commonDatabase data.Server
	maxDepth       int // maxDepth prevents the client from recursing too deeply into a document tree
}

// New creates a new instance of asContextMaker
func New(innerClient streams.Client, commonDatabase data.Server, options ...ClientOption) *Client {

	// Create the Client
	result := &Client{
		innerClient:    innerClient,
		commonDatabase: commonDatabase,
		maxDepth:       16,
	}

	// Apply options
	for _, option := range options {
		option(result)
	}

	// Pass reference down into the innerClient
	result.innerClient.SetRootClient(result)
	return result
}

func (client *Client) SetRootClient(rootClient streams.Client) {
	client.innerClient.SetRootClient(rootClient)
	client.rootClient = rootClient
}

// Load implements the streams.Client interface, and loads a document from the Interwebs.
// This method passes *most* work on to its innerClient, but does some extra work to add
// a "context" property to each document that passes it.
func (client Client) Load(uri string, options ...any) (streams.Document, error) {

	// Try to get the document from the cache
	result, err := client.innerClient.Load(uri, options...)

	if err != nil {
		return result, err
	}

	// Don't need to add context to Actors (only Documents)
	if result.IsActor() {
		return result, nil
	}

	// Don't need to add context to Collections (only Documents)
	if result.IsCollection() {
		return result, nil
	}

	// Use the Parent context in all cases (even if it's "better" than our context)
	parentContext := client.getParentContext(result)
	result.SetProperty(vocab.PropertyContext, parentContext)

	// Return modified document to the caller.
	return result, nil
}

func (client *Client) getParentContext(document streams.Document) string {

	// If this is a reply to another document...
	if inReplyTo := document.InReplyTo().ID(); inReplyTo != "" {

		// Try to load the parent...
		if parent, err := client.rootClient.Load(inReplyTo); err == nil {

			// And return its context (if any)
			if context := parent.Context(); context != "" {
				return context
			}
		}
	}

	// If this document has a context, then let's use that
	if context := document.Context(); context != "" {
		return context
	}

	// Last resort, generate an artificial context (just in case)
	return protocol_artificial + primitive.NewObjectID().Hex()
}
