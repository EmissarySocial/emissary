package asrecursor

import (
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

type Recursor struct {
	innerClient streams.Client
	maxDepth    int
}

func New(innerClient streams.Client, maxDepth int) *Recursor {

	result := &Recursor{
		innerClient: innerClient,
		maxDepth:    maxDepth,
	}

	return result
}

func (client *Recursor) Load(uri string, options ...any) (streams.Document, error) {
	result, err := client.innerClient.Load(uri, options...)

	if err != nil {
		return result, derp.Wrap(err, "asrecursor.Load", "Error loading actor from inner client")
	}

	// Actor objects
	client.recurseCollection(result, vocab.PropertyBlocked, 0)
	client.recurseCollection(result, vocab.PropertyFollowers, 0)
	client.recurseCollection(result, vocab.PropertyFollowing, 0)
	client.recurseCollection(result, vocab.PropertyLiked, 0)
	client.recurseCollection(result, vocab.PropertyOutbox, 0)

	// Document objects
	client.recurseCollection(result, vocab.PropertyContext, 0)
	client.recurseCollection(result, vocab.PropertyInReplyTo, 0)

	return result, nil
}

func (client *Recursor) recurseCollection(document streams.Document, propertyName string, depth int) {
	if collection := document.Get(propertyName); collection.NotNil() {
		for item := collection.Items(); item.NotNil(); item.Next() {
			client.recurse(item, depth+1)
		}
	}
}

func (client *Recursor) recurse(document streams.Document, depth int) {

	// TODO: HIGH: This needs to be a buffered queue or something. RN we're gong to SLAM remote servers with requests.

	// RULE: Do not exceed maxDepth
	if depth > client.maxDepth {
		return
	}

	// RULE: If "document" is only a string/id, then load it.
	if document.IsString() {
		var err error
		document, err = client.innerClient.Load(document.ID())

		if err != nil {
			return
		}
	}

	// Try to load the Actor
	if actor := document.Actor(); actor.NotNil() {
		client.recurse(actor, depth+1)
	}

	// Check InReplyTo
	if inReplyTo := document.InReplyTo(); inReplyTo.NotNil() {
		client.recurse(inReplyTo, depth+1)
	}

	// Check "Items" property for all collections
	switch document.Type() {
	case vocab.CoreTypeCollection, vocab.CoreTypeCollectionPage, vocab.CoreTypeOrderedCollection, vocab.CoreTypeOrderedCollectionPage:
		for item := document.Items(); item.NotNil(); item = item.Tail() {
			client.recurse(item.Head(), depth+1)
		}
	}
}
