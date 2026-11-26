package consumer

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SubscribeTagRelay makes the @application actor follow the relay for a tag that someone here follows
func SubscribeTagRelay(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.SubscribeTagRelay"

	if err := factory.TagRelay().Subscribe(session, args.GetString("value"), args.GetString("name")); err != nil {
		return requeue(derp.Wrap(err, location, "Subscribing to relay", args.GetString("value")))
	}

	return queue.Success()
}

// UnsubscribeTagRelay stops following a tag's relay once nobody here follows the tag
func UnsubscribeTagRelay(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.UnsubscribeTagRelay"

	if err := factory.TagRelay().Unsubscribe(session, args.GetString("value")); err != nil {
		return requeue(derp.Wrap(err, location, "Unsubscribing from relay", args.GetString("value")))
	}

	return queue.Success()
}

// BackfillTagFollow copies a tag's recent relay posts into the timeline of the User who just followed it
func BackfillTagFollow(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.BackfillTagFollow"

	userID, err := primitive.ObjectIDFromHex(args.GetString("userId"))

	if err != nil {
		return queue.Failure(derp.Wrap(err, location, "Invalid userId", args.GetString("userId")))
	}

	// RULE: Skip it if the User stopped following before this ran
	if following, err := factory.TagFollow().IsFollowing(session, userID, args.GetString("value")); err != nil {
		return requeue(derp.Wrap(err, location, "Checking follow", userID))
	} else if !following {
		return queue.Success()
	}

	if err := factory.TagRelay().Backfill(session, userID, args.GetString("name")); err != nil {
		return requeue(derp.Wrap(err, location, "Backfilling timeline", userID))
	}

	return queue.Success()
}
