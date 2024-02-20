package activitypub_user

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeCreate, vocab.Any, activityPub_CreateOrUpdate)
	inboxRouter.Add(vocab.ActivityTypeUpdate, vocab.Any, activityPub_CreateOrUpdate)
}

func activityPub_CreateOrUpdate(context Context, activity streams.Document) error {

	const location = "handler.activitypub.activityPub_CreateOrUpdate"

	// Load the actual document into the ActivityStream cache
	object := activity.UnwrapActivity()

	// Ignore these types of objects.
	switch object.Type() {

	case vocab.ObjectTypeRelationship,
		vocab.ObjectTypeProfile,
		vocab.ObjectTypePlace,
		vocab.ObjectTypeEvent,
		vocab.ObjectTypeTombstone:
		return nil
	}

	// Guarantee that we can load the object from the Interwebs.
	object, err := object.Load()

	if err != nil {
		return derp.Wrap(err, location, "Error loading activity.Object")
	}

	// Verify that this message comes from a valid "Following" object.
	followingService := context.factory.Following()
	following := model.NewFollowing()

	if err := followingService.LoadByURL(context.user.UserID, activity.Actor().ID(), &following); err != nil {
		return nil
	}

	// Try to save the message to the database (with de-duplication)
	if err := followingService.SaveMessage(&following, object, model.OriginTypePrimary); err != nil {
		return derp.Wrap(err, "handler.activitypub_receive_create", "Error saving message", context.user.UserID, object.Value())
	}

	// Success!!
	return nil
}
