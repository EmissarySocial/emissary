package asrecursor

import (
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
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

func (client *Recursor) Load(uri string, defaultValue map[string]any) (streams.Document, error) {

	result, err := client.innerClient.Load(uri, defaultValue)

	if err != nil {
		return result, derp.Wrap(err, "asrecursor.Load", "Error loading document from inner client")
	}

	if client.maxDepth > 0 {
		go client.recurse(result, 0)
	}

	result.WithOptions(streams.WithClient(client))
	return result, nil
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
		document, err = client.innerClient.Load(document.ID(), mapof.NewAny())

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
		if items := document.Items(); items.NotNil() {
			items.ForEach(func(item streams.Document) {
				client.recurse(item, depth+1)
			})
		}
	}
}
