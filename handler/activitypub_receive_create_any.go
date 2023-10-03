package handler

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeCreate, vocab.Any, activityPub_CreateOrUpdate)
	inboxRouter.Add(vocab.ActivityTypeUpdate, vocab.Any, activityPub_CreateOrUpdate)
}

func activityPub_CreateOrUpdate(factory *domain.Factory, user *model.User, document streams.Document) error {

	// Ignore these types of objects.
	switch document.Object().Type() {
	case vocab.ObjectTypeRelationship,
		vocab.ObjectTypeProfile,
		vocab.ObjectTypePlace,
		vocab.ObjectTypeEvent,
		vocab.ObjectTypeTombstone:
		return nil
	}

	followingService := factory.Following()
	following := model.NewFollowing()

	// Verify that this message comes from a "Following" object.  If not, we can cache it, but it won't become a message.
	if err := followingService.LoadByURL(user.UserID, document.Actor().ID(), &following); err != nil {
		_, _ = factory.ActivityStreams().Load(document.ID())
		return nil
	}

	// Try to save the message to the database (with de-duplication)
	if err := followingService.SaveMessage(&following, document); err != nil {
		return derp.Wrap(err, "handler.activitypub_receive_create", "Error saving message", user.UserID, document.Object().ID())
	}

	return nil
}
