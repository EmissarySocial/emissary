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

	// Locate the original "object" value as an actual object.  If the object is
	// embedded inline, use it directly.  If it is a bare URL, load it from the
	// Interwebs -- and treat a load failure as a (retryable) error, so a transient
	// network problem does not silently drop the news item.
	document := activity.UnwrapActivity()

	if document.IsString() {

		loaded, err := document.Load()

		if err != nil {
			return derp.Wrap(err, location, "Unable to load embedded object", document.Value())
		}

		document = loaded
	}

	// Gonna need the followingService in a hot sec..
	followingService := context.factory.Following()
	following := model.NewFollowing()

	// If the "Following" record cannot be found, then do not add a message
	if err := followingService.LoadByURL(context.session, context.user.UserID, activity.ActorID(), &following); err != nil {
		return derp.Wrap(err, location, "Unable to locate `Following` record", context.user.UserID)
	}

	// Try to save the message to a folder (with de-duplication)
	if err := followingService.SaveNewsItem(context.session, &following, document, model.OriginTypePrimary); err != nil {
		return derp.Wrap(err, location, "Unable to save news item", context.user.UserID, activity.Value())
	}

	// Add this document to a context (if necessary)
	if hasLocalReplyOrContext(document, context.factory.Host()) {

		context.factory.Queue().NewTask(
			"AddToCollection",
			mapof.Any{
				"host":    context.factory.Hostname(), // The host that received the activity
				"userId":  context.user.UserID.Hex(),  // The user who received the activity
				"actorId": activity.ActorID(),         // The actor adding the item to the context
				"url":     document.ID(),              // The URL to add to a context collection
			},
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
