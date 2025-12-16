package consumer

import (
	"net/url"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func MoveUser(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.MoveUser"

	// Collect target Actor
	newActorURL := args.GetString("actor")

	// Validate that Actor URL is valid
	if newActorURL == "" {
		return queue.Failure(derp.BadRequest(location, "Actor URL is required"))
	}

	if _, err := url.Parse(newActorURL); err != nil {
		return queue.Failure(derp.BadRequest(location, "Actor URL is not a valid URL", newActorURL))
	}

	// Collect Oracle value to forward new requests to later.
	movedTo := args.GetString("oracle")

	// Collect UserID
	userID, err := primitive.ObjectIDFromHex(args.GetString("userId"))

	if err != nil {
		return queue.Failure(derp.Internal(location, "UserID must be a valid ObjectID", args))
	}

	// Load the User from the database
	user := model.NewUser()
	if err := factory.User().LoadByID(session, userID, &user); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to load User"))
	}

	// Send `Move` message to all followers
	followers := factory.Follower().RangeByUserID(session, userID)

	for follower := range followers {
		queue.NewTask("SendActivityPubMessage", mapof.Any{
			"actorType": "User",
			"actorID":   user.UserID,
			"to":        follower.Actor.ProfileURL,
			"message": mapof.Any{
				vocab.PropertyActor:  user.ActivityPubURL(),
				vocab.PropertyType:   vocab.ActivityTypeMove,
				vocab.PropertyObject: user.ActivityPubURL(),
				vocab.PropertyTarget: newActorURL,
			},
		})
	}

	// Mark related Streams as `MovedTo`
	if err := factory.Stream().MoveByUserID(session, userID, movedTo); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to delete related streams"))
	}

	// Delete related Outbox Messages
	if err := factory.Outbox().DeleteByParentID(session, model.ActorTypeUser, userID); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to delete related outbox messages"))
	}

	// Delete related Following
	if err := factory.Following().DeleteByUserID(session, userID, "moved"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to delete related following records"))
	}

	// Delete related Folders
	if err := factory.Folder().DeleteByUserID(session, userID, "moved"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to delete related Folders"))
	}

	// Delete related Inbox Messages
	if err := factory.Inbox().DeleteByUserID(session, userID, "moved"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to delete related inbox messages"))
	}

	// Delete related Conversations
	if err := factory.Conversation().DeleteByUserID(session, userID, "moved"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to delete related conversations"))
	}

	// Delete related Rules
	if err := factory.Rule().DeleteByUserID(session, userID, "moved"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to delete related rules"))
	}

	// Delete related Followers
	if err := factory.Follower().DeleteByUserID(session, userID, "moved"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to delete related Followers"))
	}

	// Delete related Annotations
	if err := factory.Annotation().DeleteByUserID(session, userID, "moved"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to delete related annotations"))
	}

	// Delete related Merchant Accounts
	if err := factory.MerchantAccount().DeleteByUserID(session, userID, "moved"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to delete related merchant accounts"))
	}

	// Delete related Products
	if err := factory.Product().DeleteByUserID(session, userID, "moved"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to delete related products"))
	}

	// Delete related Circles
	if err := factory.Circle().DeleteByUserID(session, userID, "moved"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to delete related circles"))
	}

	// Delete related Privileges
	if err := factory.Privilege().DeleteByUserID(session, userID, "moved"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to delete related privileges"))
	}

	// Delete related Mentions
	if err := factory.Mention().DeleteByObjectID(session, model.MentionTypeUser, userID, "moved"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to delete related mentions"))
	}

	// Delete related Responses
	if err := factory.Response().DeleteByUserID(session, userID, "moved"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to delete related responses"))
	}

	return queue.Success()

}
