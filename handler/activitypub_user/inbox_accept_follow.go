package activitypub_user

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/first"
)

// init registers the handler for inbound Accept/Follow activities
func init() {
	inboxRouter.Add(vocab.ActivityTypeAccept, vocab.ActivityTypeFollow, inbox_AcceptFollow)
}

// This funciton handles ActivityPub "Accept/Follow" activities, meaning that
// it is called with a remote server accepts our follow request.
func inbox_AcceptFollow(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.inbox_AcceptFollow"

	followingService := context.factory.Following()

	// Parse the Object.ID of the activity, which should be our original "Follow" activity.  Passing
	// our hostname binds it to THIS domain: the remote server is echoing a URL back to us, and only
	// a URL we actually issued can name a local `Following` record.
	userID, followingID, err := service.ParseProfileURL_AsFollowing(context.factory.Hostname(), activity.Object().ID())

	if err != nil {
		return derp.Wrap(err, location, "Parsing followingID", activity.Object().ID())
	}

	// Try to load the original `Following` record.
	// If it doesn't already exist, then this message is invalid.
	following := model.NewFollowing()
	if err := followingService.LoadByID(context.session, userID, followingID, &following); err != nil {
		return derp.Wrap(err, location, "Loading `Following` record", userID, followingID)
	}

	// RULE: Validate that the `Following` actor matches the `Accept` actor
	if following.ProfileURL != activity.ActorID() {
		return derp.Forbidden(location, "Invalid `Accept` transaction", following.ProfileURL, activity.ActorID())
	}

	// Populate our `Following` record with the NAME and AVATAR of the remote actor
	remoteActor, err := activity.Actor().Load()

	if err != nil {
		return derp.Wrap(err, location, "Loading remote actor", activity.Actor())
	}

	// Upgrade the `Following` record to ActivityPub
	following.Label = remoteActor.Name()
	following.IconURL = first.String(remoteActor.IconOrImage().URL(), following.IconURL)
	following.Method = model.FollowingMethodActivityPub
	following.Secret = ""
	following.PollDuration = 30

	// Save the `Following` record to the database
	if err := followingService.SetStatusSuccess(context.session, &following); err != nil {
		return derp.Wrap(err, location, "Saving `Following` document.", following)
	}

	return nil
}
