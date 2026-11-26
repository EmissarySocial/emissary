package activitypub_application

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// init registers the handler for a relay announcing a tagged post
func init() {
	inboxRouter.Add(vocab.ActivityTypeAnnounce, vocab.Any, func(context Context, activity streams.Document) error {

		const location = "handler.activitypub_application.ReceiveAnnounce"

		// RULE: Only relays the @application actor follows may hand it posts
		tagRelay := model.NewTagRelay()
		if err := context.factory.TagRelay().LoadByActorURL(context.session, activity.ActorID(), &tagRelay); err != nil {
			if derp.IsNotFound(err) {
				return nil
			}
			return derp.Wrap(err, location, "Loading TagRelay", activity.ActorID())
		}

		// Load the announced post, which also puts it in the shared cache for delivery
		post, err := activity.Object().Load()

		if err != nil {
			return derp.Wrap(err, location, "Loading announced post", activity.Object().ID())
		}

		// Offer the post to everyone here who follows one of its #hashtags
		context.factory.TagFollow().QueueDelivery(context.session, post, primitive.NilObjectID)
		return nil
	})
}
