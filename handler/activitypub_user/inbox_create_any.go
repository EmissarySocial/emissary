package activitypub_user

import (
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
)

func init() {

	// Wildcard to handle Create/Update of (nearlly) any type
	inboxRouter.Add(vocab.ActivityTypeCreate, vocab.Any, inbox_CreateOrUpdate)
	inboxRouter.Add(vocab.ActivityTypeUpdate, vocab.Any, inbox_CreateOrUpdate)

	// These values are skipped
	inboxRouter.Add(vocab.ActivityTypeCreate, vocab.ObjectTypeRelationship, inbox_Unknown)
	inboxRouter.Add(vocab.ActivityTypeCreate, vocab.ObjectTypeProfile, inbox_Unknown)
	inboxRouter.Add(vocab.ActivityTypeCreate, vocab.ObjectTypeTombstone, inbox_Unknown)

	inboxRouter.Add(vocab.ActivityTypeUpdate, vocab.ObjectTypeRelationship, inbox_Unknown)
	inboxRouter.Add(vocab.ActivityTypeUpdate, vocab.ObjectTypeProfile, inbox_Unknown)
	inboxRouter.Add(vocab.ActivityTypeUpdate, vocab.ObjectTypeTombstone, inbox_Unknown)
}

func inbox_CreateOrUpdate(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.inbox_CreateOrUpdate"

	// RULE: No additional processing for non-public activites. These have already
	// been added to the User's inbox (in inbox_SaveActivity) so they'll be picked
	// up by the chat client without any further action.
	if activity.NotPublic() {
		return nil
	}

	// Load the original document directly from the Interwebs.
	document, err := activity.UnwrapActivity().Load()

	if err != nil {
		return derp.Wrap(err, location, "Unable to load enbedded object")
	}

	// Gonna need the followingService in a hot sec..
	followingService := context.factory.Following()
	following := model.NewFollowing()

	// If the "Following" record cannot be found, then do not add a message
	if err := followingService.LoadByURL(context.session, context.user.UserID, activity.Actor().ID(), &following); err != nil {
		return derp.Wrap(err, location, "Unable to locate `Following` record", context.user.UserID)
	}

	// Try to save the message to a folder (with de-duplication)
	if err := followingService.SaveNewsItem(context.session, &following, document, model.OriginTypePrimary); err != nil {
		return derp.Wrap(err, location, "Unable to save news item", context.user.UserID, activity.Value())
	}

	// Add this document to a context (if necessary)
	if hasLocalReplyOrContext(document, context.factory.Host()) {

		context.factory.Queue().NewTask(
			"AddToContext",
			mapof.Any{"url": document.ID()},
		)
	}

	// Success!
	return nil
}

// hasLocalReplyOrContext returns TRUE if the document belongs to a context that
// is owned by this server, or replies to a document that is owned by this server.
func hasLocalReplyOrContext(document streams.Document, host string) bool {

	if documentContext := document.Context(); strings.HasPrefix(documentContext, host) {
		return true
	}

	if inReplyTo := document.InReplyTo().ID(); strings.HasPrefix(inReplyTo, host) {
		return true
	}

	return false
}
