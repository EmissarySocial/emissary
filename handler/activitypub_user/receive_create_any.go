package activitypub_user

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeCreate, vocab.Any, receive_CreateOrUpdate)
	inboxRouter.Add(vocab.ActivityTypeUpdate, vocab.Any, receive_CreateOrUpdate)
}

func receive_CreateOrUpdate(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.receive_CreateOrUpdate"

	// Load the actual document into the ActivityStream cache
	object := activity.UnwrapActivity()

	// Ignore these types of objects.
	switch object.Type() {

	case vocab.ObjectTypeRelationship,
		vocab.ObjectTypeProfile,
		vocab.ObjectTypeTombstone:
		return nil
	}

	// Guarantee that we can load the object from the Interwebs.
	if _, err := object.Load(); err != nil {
		return derp.Wrap(err, location, "Error loading activity.Object")
	}

	// Try to add a message to the User's inbox
	if err := saveMessage(context, activity, activity.Actor().ID(), model.OriginTypePrimary); err != nil {
		return derp.Wrap(err, location, "Error saving message", context.user.UserID, activity.Value())
	}

	// Success!!
	return nil
}
