package activitypub_user

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeFollow, vocab.Any, func(context Context, activity streams.Document) error {

		const location = "handler.activitypub_user.inbox_FollowAny"

		// Look up the requested user account
		userService := context.factory.User()

		// Try to verify the User.  Passing our hostname binds the Follow's object to THIS domain --
		// the path grammar alone would accept `https://anywhere.example/@<localUserID>`.
		userID, err := service.ParseProfileURL_UserID(context.factory.Hostname(), activity.Object().ID())

		if err != nil {
			return derp.Wrap(err, location, "Invalid User URL", activity.Object().ID())
		}

		// RULE: A Follow must be delivered to the inbox of the actor it names.  This is a 422, not a
		// 500: the request is well-formed and the server is healthy -- the sender simply addressed a
		// Follow for one actor to a different actor's inbox, and no retry will change that.
		if userID != context.user.UserID {
			return derp.Validation("Follow object does not match the User who owns this inbox", userID.Hex(), context.user.UserID.Hex(), derp.WithLocation(location))
		}

		// RULE: A blocked actor may not Follow. Verify first (D5 exception set), then reject loudly --
		// the check lives HERE because the wire gate deliberately defers Follows; see
		// activitypub.IsWireGateException (handler/activitypub/ruleValidator.go) for why Follow
		// rejects loudly while other activity types are silently discarded.
		blocked, err := context.factory.Rule().IsActorBlocked(context.session, context.user.UserID, activity)

		if err != nil {
			return derp.Wrap(err, location, "Checking block rules", activity.ActorID())
		}

		if blocked {
			return derp.Forbidden(location, "Blocked by rule", activity.Object().ID())
		}

		// Try to look up the complete actor record from the activity
		document, err := activity.Actor().Load()

		if err != nil {
			return derp.Wrap(err, location, "Parsing actor", activity)
		}

		// Try to create a new follower record
		followerService := context.factory.Follower()
		follower := model.NewFollower()
		if err := followerService.NewActivityPubFollower(context.session, model.FollowerTypeUser, context.user.UserID, document, &follower); err != nil {
			return derp.Wrap(err, location, "Creating new follower", context.user)
		}

		// Try to load the Actor for this user
		actor, err := userService.ActivityPubActor(context.session, context.user.UserID)

		if err != nil {
			return derp.Wrap(err, location, "Loading actor", context.user)
		}

		// Send the "Accept" message to the Requester as a post-commit queue task (F3): the signed
		// delivery happens after this transaction commits and is independently retryable.
		acceptID := followerService.ActivityPubID(&follower)
		context.factory.Outbox().SendAccept(context.session, actor.ActorID(), acceptID, activity)

		// Create a FOLLOW notification for the recipient.  This lives here (rather than in the
		// central NotifyFromActivity hook) because it must fire only AFTER the Follow is validated
		// and accepted.  A notification failure must not fail the follow, so report-and-continue.
		if err := context.factory.Notification().NotifyFollow(context.session, context.user, activity); err != nil {
			derp.Report(derp.Wrap(err, location, "Creating follow notification", context.user.UserID))
		}

		// Voila!
		return nil
	})
}
