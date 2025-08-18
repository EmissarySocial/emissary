package ascontextmaker

import (
	"strings"

	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/davecgh/go-spew/spew"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// asContextMaker is a hannibal.Streams middleware that adds a "context" property to all documents
// based on their "InReplyTo" property.  If a document does not have a context or inReplyTo, then
// it is its own context, and is updated to reflect that.
type Client struct {
	rootClient  streams.Client
	innerClient streams.Client
	enqueue     chan<- queue.Task
	maxDepth    int // maxDepth prevents the client from recursing too deeply into a document tree
}

// New creates a new instance of asContextMaker
func New(innerClient streams.Client, enqueue chan<- queue.Task, options ...ClientOption) *Client {

	// Create the Client
	result := &Client{
		innerClient: innerClient,
		enqueue:     enqueue,
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

	// Calculate the best context to use for this document
	spew.Dump("contextMaker.Load ---------------------------------")

	myContext := client.getContext(result)
	parentContext := client.getParentContext(result.InReplyTo().ID())

	spew.Dump(result.InReplyTo().ID(), parentContext)
	bestContext := client.getBestContext(myContext, parentContext)

	// Update context(s) if necessary
	if bestContext != parentContext {
		client.enqueue <- queue.NewTask(
			"UpdateContext",
			mapof.Any{
				"oldContext": parentContext,
				"newContext": bestContext,
			},
			queue.WithPriority(128),
		)
	}

	if bestContext != myContext {
		result.SetProperty(vocab.PropertyContext, bestContext)
	}

	/*
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
	*/

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

func (client *Client) getContext(document streams.Document) string {

	// Use the context property if it exists
	if context := document.Context(); context != "" {
		return context
	}

	// Otherwise, none
	return ""
}

func (client *Client) getParentContext(inReplyTo string) string {

	// If this is a reply to another document...
	if inReplyTo != "" {

		// Try to load the parent...
		if parent, err := client.rootClient.Load(inReplyTo); err == nil {

			// And return its context (if any)
			if context := parent.Context(); context != "" {
				return context
			}
		}
	}

	// Last resort, generate an artificial context (just in case)
	return protocol_artificial + primitive.NewObjectID().Hex()
}

func (client *Client) getBestContext(myContext string, parentContext string) string {

	spew.Dump("contextMaker.getBestContext:::", myContext, parentContext)

	// Shortcut if they're the same
	if myContext == parentContext {
		return myContext
	}

	// First, use any context that's an actual URL, preferring parent context first
	if strings.HasPrefix(parentContext, protocol_https) {
		return parentContext
	}

	if strings.HasPrefix(parentContext, protocol_http) {
		return parentContext
	}

	if strings.HasPrefix(myContext, protocol_https) {
		return myContext
	}

	if strings.HasPrefix(myContext, protocol_http) {
		return myContext
	}

	// Next, choose ANYTHING BUT an artificial://
	if strings.HasPrefix(myContext, protocol_artificial) {
		if parentContext != "" {
			return parentContext
		}
		return myContext
	}

	if strings.HasPrefix(parentContext, protocol_artificial) {
		if myContext != "" {
			return myContext
		}
		return parentContext
	}

	// If we have ANYTHING AT ALL in the "parent" context
	// then prefer that over whatever is left in "my" context
	if parentContext != "" {
		return parentContext
	}

	// use "my" context, even if it's empty string.
	return myContext
}
