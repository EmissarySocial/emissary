package consumer

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MailingListAddMember writes a confirmed EMAIL Follower into their User's mailing list
func MailingListAddMember(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.MailingListAddMember"

	userID, err := primitive.ObjectIDFromHex(args.GetString("userId"))

	if err != nil {
		return queue.Failure(derp.Wrap(err, location, "Invalid userId", args))
	}

	followerID, err := primitive.ObjectIDFromHex(args.GetString("followerId"))

	if err != nil {
		return queue.Failure(derp.Wrap(err, location, "Invalid followerId", args))
	}

	// Re-read the Follower, so that a task delayed behind others pushes current values
	follower := model.NewFollower()

	if err := factory.Follower().LoadByID(session, userID, followerID, &follower); err != nil {

		// A Follower deleted between the enqueue and the run has nothing to push, and the
		// removal task that deleted them is already in the queue behind this one.
		if derp.IsNotFound(err) {
			return queue.Success()
		}

		return queue.Error(derp.Wrap(err, location, "Loading Follower", args))
	}

	// RULE: re-check the state here, not only at enqueue. A Follower paused by a block rule
	// between the two must not be handed to a third party (D20).
	if follower.StateID != model.FollowerStateActive {
		return queue.Success()
	}

	if err := factory.UserConnection().MailchimpAddMember(session, userID, &follower); err != nil {
		return requeue(derp.Wrap(err, location, "Adding member to mailing list", args))
	}

	// Come with me if you want to be emailed
	return queue.Success()
}
