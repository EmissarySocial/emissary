package activitypub_user

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {

	// Wildcard to handle Create/Update of (nearlly) any type
	inboxRouter.Add(vocab.ActivityTypeCreate, vocab.Any, receive_CreateOrUpdate)
	inboxRouter.Add(vocab.ActivityTypeUpdate, vocab.Any, receive_CreateOrUpdate)

	// These values are skipped
	inboxRouter.Add(vocab.ActivityTypeCreate, vocab.ObjectTypeRelationship, receive_Unknown)
	inboxRouter.Add(vocab.ActivityTypeCreate, vocab.ObjectTypeProfile, receive_Unknown)
	inboxRouter.Add(vocab.ActivityTypeCreate, vocab.ObjectTypeTombstone, receive_Unknown)

	inboxRouter.Add(vocab.ActivityTypeUpdate, vocab.ObjectTypeRelationship, receive_Unknown)
	inboxRouter.Add(vocab.ActivityTypeUpdate, vocab.ObjectTypeProfile, receive_Unknown)
	inboxRouter.Add(vocab.ActivityTypeUpdate, vocab.ObjectTypeTombstone, receive_Unknown)
}

func receive_CreateOrUpdate(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.receive_CreateOrUpdate"

	// Collect the actorID from the Activity
	actorID := activity.Actor().ID()

	if actorID == "" {
		return derp.BadRequestError(location, "Activity must have an ActorID", activity.Value())
	}

	// Load the actual document into the ActivityStream cache
	embeddedObject := activity.UnwrapActivity()

	// Load the original document directly from the Interwebs.
	document, err := embeddedObject.Load()

	if err != nil {
		return derp.Wrap(err, location, "Unable to load enbedded object")
	}

	// Gonna need the followingService in a hot sec..
	followingService := context.factory.Following()

	/* TEMPORARILY REMOVING "DIRECT MESSAGES" BECUASE THE UX IS NOT READY.
	// If the user is "mentioned" then save the message to their direct messages
	if userIsMentioned(context.user, document) {

		if err := followingService.SaveDirectMessage(context.session, context.user, activity); err != nil {
			return derp.Wrap(err, location, "Unable to save direct message", context.user.UserID, activity.Value())
		}
		return nil
	}
	*/

	// Verify that this message comes from a valid "Following" object.
	following := model.NewFollowing()

	// If the "Following" record cannot be found, then do not add a message
	if err := followingService.LoadByURL(context.session, context.user.UserID, actorID, &following); err != nil {
		return derp.Wrap(err, location, "Unable to locate `Following` record", context.user.UserID, actorID)
	}

	// Try to save the message to a folder (with de-duplication)
	if err := followingService.SaveMessage(context.session, &following, document, model.OriginTypePrimary); err != nil {
		return derp.Wrap(err, location, "Unable to save message", context.user.UserID, activity.Value())
	}

	return nil
}

/*
// userIsMentioned returns TRUE if this user is tagged in the document
func userIsMentioned(user *model.User, document streams.Document) bool {

	actorID := user.ActivityPubURL()

	for mention := range document.Tag().Range() {

		if mention.Type() == vocab.LinkTypeMention {
			if mention.ID() == actorID {
				return true
			}
		}
	}

	return false
}
*/
