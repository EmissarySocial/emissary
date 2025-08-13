package ascontextmaker

import (
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// asContextMaker is a hannibal.Streams middleware that adds a "context" property to all documents
// based on their "InReplyTo" property.  If a document does not have a context or inReplyTo, then
// it is its own context, and is updated to reflect that.
type Client struct {
	rootClient  streams.Client
	innerClient streams.Client
	maxDepth    int // maxDepth prevents the client from recursing too deeply into a document tree
}

// New creates a new instance of asContextMaker
func New(innerClient streams.Client, options ...ClientOption) *Client {

	// Create the Client
	result := &Client{
		innerClient: innerClient,
		maxDepth:    16,
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
	client.rootClient = rootClient
	client.innerClient.SetRootClient(rootClient)
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

	// If the document already has a context property, then
	// there is nothing more to add
	if context := result.Context(); context != "" {
		return result, nil
	}

	// If there is an ostatus:conversation property, then use that
	if conversation := result.Get("conversation"); conversation.NotNil() {
		result.SetProperty(vocab.PropertyContext, conversation)
		return result, nil
	}

	// Past here, we really WANT a context, so let's make a default to start with...
	result.SetProperty(vocab.PropertyContext, "artificialcontext://"+result.ID())

	// ... then try to find something better than this.

	// Check configuration rules (re: history and duplicates)
	// If no more recursion is allowed, then simply stop here.
	config := NewLoadConfig(options...)

	if client.NotAllowed(uri, config) {
		return result, nil
	}

	// If we have an "inReplyTo" field, then try to load that value
	// to use/generate its context
	if result.InReplyTo().NotNil() {

		options = append(options, WithHistory(uri))

		for inReplyTo := result.InReplyTo(); inReplyTo.NotNil(); inReplyTo = inReplyTo.Tail() {
			if parent, err := client.rootClient.Load(inReplyTo.ID(), options...); err == nil {
				if context := parent.Context(); context != "" {
					result.SetProperty(vocab.PropertyContext, context)
					break
				}
			}
		}
	}

	// Return modified document to the caller.
	return result, nil
}

// IsAllowed returns TRUE if the client rules allow the provided URI to be loaded
func (client *Client) IsAllowed(uri string, config LoadConfig) bool {

	// If the history is already too long, then stop no matter what
	if len(config.history) >= client.maxDepth {
		return false
	}

	// Search the history for this URI.  If it's already been loaded, then stop.
	for _, value := range config.history {
		if value == uri {
			return false
		}
	}

	// This URI is allowed to be loaded
	return true
}

// NotAllowed returns TRUE if the client rules DO NOT allow the provided URI to be loaded
func (client *Client) NotAllowed(uri string, config LoadConfig) bool {
	return !client.IsAllowed(uri, config)
}
