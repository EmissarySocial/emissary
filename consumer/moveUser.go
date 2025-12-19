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
)

// MoveUser is a background task that finalizes moving a user to a new server.
// It issues a `Move` message to the target server and to all followers, then deletes or updates
// all related records in the local database.
func MoveUser(factory *service.Factory, session data.Session, user *model.User, args mapof.Any) queue.Result {

	const location = "consumer.MoveUser"

	/******************************************
	 * 1: Collect Prerequisite Data
	 ******************************************/

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

	/******************************************
	 * 2: Send a `Move` to the Target Actor
	 ******************************************/

	// Send a `Move` message to the target server
	factory.Queue().NewTask("SendActivityPubMessage", mapof.Any{
		"host":      factory.Hostname(),
		"actorType": model.ActorTypeUser,
		"actorID":   user.UserID,
		"to":        newActorURL,
		"message": mapof.Any{
			vocab.AtContext:      vocab.ContextTypeActivityStreams,
			vocab.PropertyTo:     newActorURL,
			vocab.PropertyActor:  user.ActivityPubURL(),
			vocab.PropertyType:   vocab.ActivityTypeMove,
			vocab.PropertyObject: user.ActivityPubURL(),
			vocab.PropertyTarget: newActorURL,
		},
	})

	/******************************************
	 * 3. Send a `Move` to all Followers
	 ******************************************/

	// Send `Move` message to all followers
	followers := factory.Follower().RangeByUserID(session, user.UserID)

	for follower := range followers {
		queue.NewTask("SendActivityPubMessage", mapof.Any{
			"host":      factory.Hostname(),
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

	/******************************************
	 * 3. Update/Delete Records in Profile
	 ******************************************/

	// Mark related Streams as `MovedTo`
	if err := factory.Stream().MoveByUserID(session, user.UserID, movedTo); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to delete related Streams"))
	}

	// Delete related Annotations
	if err := factory.Annotation().DeleteByUserID(session, user.UserID, "moved"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to delete related Annotations"))
	}

	// Delete related Outbox Messages
	if err := factory.Outbox().DeleteByParentID(session, model.ActorTypeUser, user.UserID); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to delete related Outbox messages"))
	}

	// Delete related Inbox Messages
	if err := factory.Inbox().DeleteByUserID(session, user.UserID, "moved"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to delete related Inbox messages"))
	}

	// Delete related Conversations
	if err := factory.Conversation().DeleteByUserID(session, user.UserID, "moved"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to delete related Conversations"))
	}

	// Delete related Folders
	if err := factory.Folder().DeleteByUserID(session, user.UserID, "moved"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to delete related Folders"))
	}

	// Delete related Rules
	if err := factory.Rule().DeleteByUserID(session, user.UserID, "moved"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to delete related Rules"))
	}

	// Delete related Following
	if err := factory.Following().DeleteByUserID(session, user.UserID, "moved"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to delete related Following records"))
	}

	// Delete related Followers
	if err := factory.Follower().DeleteByUserID(session, user.UserID, "moved"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to delete related Followers"))
	}

	// Delete related Privileges
	if err := factory.Privilege().DeleteByUserID(session, user.UserID, "moved"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to delete related Privileges"))
	}

	// Delete related Products
	if err := factory.Product().DeleteByUserID(session, user.UserID, "moved"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to delete related Products"))
	}

	// Delete related Merchant Accounts
	if err := factory.MerchantAccount().DeleteByUserID(session, user.UserID, "moved"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to delete related MerchantAccounts"))
	}

	// Delete related Circles
	if err := factory.Circle().DeleteByUserID(session, user.UserID, "moved"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to delete related Circles"))
	}

	// Delete related Mentions
	if err := factory.Mention().DeleteByObjectID(session, model.MentionTypeUser, user.UserID, "moved"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to delete related Mentions"))
	}

	// Delete related Responses
	if err := factory.Response().DeleteByUserID(session, user.UserID, "moved"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to delete related Responses"))
	}

	// Woot.
	return queue.Success()
}
