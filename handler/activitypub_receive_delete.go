package handler

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeDelete, vocab.Any, func(factory *domain.Factory, user *model.User, document streams.Document) error {

		object := document.Object()

		// RULE: Ignore these Object Types
		switch object.Type() {
		case vocab.ObjectTypeRelationship,
			vocab.ObjectTypeProfile,
			vocab.ObjectTypePlace,
			vocab.ObjectTypeEvent,
			vocab.ObjectTypeTombstone:
			return nil
		}

		inboxService := factory.Inbox()
		message := model.NewMessage()

		// Look for the original message that's being deleted
		if err := inboxService.LoadByURL(user.UserID, object.ID(), &message); err != nil {
			return derp.Wrap(err, "handler.activitypub_receive_delete", "Error loading message", user.UserID, object.ID())
		}

		// TODO: HIGH: Validate that the document.Actor is the same as the message.Origin

		// Delete the original messag
		if err := inboxService.Delete(&message, "Deleted via ActivityPub"); err != nil {
			return derp.Wrap(err, "handler.activitypub_receive_delete", "Error deleting message", user.UserID, object.ID())
		}

		// Who let the dogs out?
		return nil
	})
}
