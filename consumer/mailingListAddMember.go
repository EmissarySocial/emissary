package consumer

import (
	"fmt"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MailingListAddMember writes a confirmed EMAIL Follower into their User's mailing list
func MailingListAddMember(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.MailingListAddMember"

	// TEMPORARY (Mailchimp sync diagnosis): remove once the sync is confirmed working
	log.Info().Str("trace", "MailchimpTrace").Str("step", "4-consumer:enter").
		Interface("args", args).
		Str("userIdType", fmt.Sprintf("%T", args["userId"])).
		Str("followerIdType", fmt.Sprintf("%T", args["followerId"])).
		Str("userIdString", args.GetString("userId")).
		Str("followerIdString", args.GetString("followerId")).
		Str("factoryHostname", factory.Hostname()).
		Msg("MailchimpTrace: consumer MailingListAddMember started")

	userID, err := primitive.ObjectIDFromHex(args.GetString("userId"))

	if err != nil {
		log.Info().Str("trace", "MailchimpTrace").Str("step", "4-consumer:badUserId").Str("error", derp.Serialize(err)).Msg("MailchimpTrace: invalid userId")
		return queue.Failure(derp.Wrap(err, location, "Invalid userId", args))
	}

	followerID, err := primitive.ObjectIDFromHex(args.GetString("followerId"))

	if err != nil {
		log.Info().Str("trace", "MailchimpTrace").Str("step", "4-consumer:badFollowerId").Str("error", derp.Serialize(err)).Msg("MailchimpTrace: invalid followerId")
		return queue.Failure(derp.Wrap(err, location, "Invalid followerId", args))
	}

	// Re-read the Follower, so that a task delayed behind others pushes current values
	follower := model.NewFollower()

	if err := factory.Follower().LoadByID(session, userID, followerID, &follower); err != nil {

		// A Follower deleted between the enqueue and the run has nothing to push, and the
		// removal task that deleted them is already in the queue behind this one.
		if derp.IsNotFound(err) {
			log.Info().Str("trace", "MailchimpTrace").Str("step", "4-consumer:followerNotFound").
				Str("userId", userID.Hex()).Str("followerId", followerID.Hex()).Str("error", derp.Serialize(err)).
				Msg("MailchimpTrace: Follower NOT FOUND by (userId, followerId); task ends as SUCCESS and nothing is pushed")
			return queue.Success()
		}

		log.Info().Str("trace", "MailchimpTrace").Str("step", "4-consumer:followerLoadError").Str("error", derp.Serialize(err)).
			Msg("MailchimpTrace: error loading Follower; task will be retried")
		return queue.Error(derp.Wrap(err, location, "Loading Follower", args))
	}

	log.Info().Str("trace", "MailchimpTrace").Str("step", "4-consumer:followerLoaded").
		Str("followerId", follower.FollowerID.Hex()).Str("parentId", follower.ParentID.Hex()).
		Str("parentType", follower.ParentType).Str("method", follower.Method).Str("stateId", follower.StateID).
		Str("emailAddress", follower.Actor.EmailAddress).Str("name", follower.Actor.Name).
		Msg("MailchimpTrace: Follower re-loaded inside the consumer")

	// RULE: re-check the state here, not only at enqueue. A Follower blocked by a rule
	// between the two must not be handed to a third party (D20).
	if follower.StateID != model.FollowerStateActive {
		log.Info().Str("trace", "MailchimpTrace").Str("step", "4-consumer:notActive").Str("stateId", follower.StateID).
			Msg("MailchimpTrace: Follower is not ACTIVE in the consumer; task ends as SUCCESS and nothing is pushed")
		return queue.Success()
	}

	if err := factory.UserConnection().MailchimpAddMember(session, userID, &follower); err != nil {
		result := requeue(derp.Wrap(err, location, "Adding member to mailing list", args))
		log.Info().Str("trace", "MailchimpTrace").Str("step", "4-consumer:addMemberError").
			Int("errorCode", derp.ErrorCode(err)).Str("resultStatus", result.Status).Str("error", derp.Serialize(err)).
			Msg("MailchimpTrace: MailchimpAddMember returned an error")
		return result
	}

	log.Info().Str("trace", "MailchimpTrace").Str("step", "4-consumer:done").Str("followerId", followerID.Hex()).
		Msg("MailchimpTrace: MailchimpAddMember returned no error; task SUCCESS (see step 5 logs for whether anything was actually sent)")

	// Come with me if you want to be emailed
	return queue.Success()
}
