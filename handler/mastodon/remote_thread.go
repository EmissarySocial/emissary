package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/toot/object"
)

// documentAuthorURL returns the address of a post's author, taken from attributedTo or else actor.
func documentAuthorURL(document streams.Document) string {

	if authorURL := document.AttributedTo().ID(); authorURL != "" {
		return authorURL
	}

	return document.ActorID()
}

// remoteReplyIDs returns the status ID and the author's account ID of the post a document replies to,
// or the status ID alone when the parent cannot be read.
func remoteReplyIDs(client streams.Client, factory *service.Factory, session data.Session, document streams.Document) (string, string) {

	parentURL := document.InReplyTo().ID()

	if parentURL == "" {
		return "", ""
	}

	// A parent on this server is a Stream, whose author is already known
	parent := model.NewStream()

	if err := factory.Stream().LoadByURL(session, parentURL, &parent); err == nil {
		return replyIDsFor(parentURL, &parent)
	}

	// Any other parent has to be read from its own server to learn who wrote it
	parentDocument, err := client.Load(parentURL)

	// RULE: clients show "replying to" only when given both IDs, so a missing author hides the line
	if err != nil {
		return model.EncodeRemoteStatusID(parentURL), ""
	}

	authorURL := documentAuthorURL(parentDocument)

	if authorURL == "" {
		return model.EncodeRemoteStatusID(parentURL), ""
	}

	return model.EncodeRemoteStatusID(parentURL), resolveAccountID(factory, session, authorURL)
}

// applyRemoteReply fills in what a post from another server replies to.
func applyRemoteReply(status *object.Status, client streams.Client, factory *service.Factory, session data.Session, document streams.Document) {
	status.InReplyToID, status.InReplyToAccountID = remoteReplyIDs(client, factory, session, document)
}
