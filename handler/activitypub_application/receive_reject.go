package activitypub_application

import (
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// init registers the handler for a relay rejecting the @application actor's Follow
func init() {
	inboxRouter.Add(vocab.ActivityTypeReject, vocab.Any, func(context Context, activity streams.Document) error {

		// The object is our Follow (embedded or as its ID); an answer to some other Follow passes quietly
		err := context.factory.TagRelay().Reject(context.session, activity.Object().ID(), activity.ActorID())

		if derp.IsNotFound(err) {
			return nil
		}

		if err != nil {
			return derp.Wrap(err, "handler.activitypub_application.ReceiveReject", "Removing rejected relay", activity.Object().ID())
		}

		return nil
	})
}
