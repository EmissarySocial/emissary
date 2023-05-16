package handler

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeLike, vocab.Any, func(factory *domain.Factory, user *model.User, activity streams.Document) error {

		inboxService := factory.Inbox()

		object, err := activity.Object().Load()

		if err != nil {
			return derp.Wrap(err, "activitypub.handler.ActivityPubRouter", "Unable to load JSON-LD Object", activity.Object())
		}

		// If we already have the object of the Like/Dislike, then increment counters
		objectID := object.ID()
		inboxMessage := model.NewMessage()
		if err := inboxService.LoadByURL(user.UserID, objectID, &inboxMessage); err == nil {
			inboxMessage.Responses.LikeCount++

			if err := inboxService.Save(&inboxMessage, "Incremented Like Count"); err != nil {
				return err
			}

			return nil
		}

		// If not, then try to load the object from JSON-LD and add to the inbox

		// If not, then

		return nil
	})
}
