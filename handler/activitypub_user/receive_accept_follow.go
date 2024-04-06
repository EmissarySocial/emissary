package activitypub_user

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/first"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeAccept, vocab.ActivityTypeFollow, receive_AcceptFollow)
}

// This funciton handles ActivityPub "Accept/Follow" activities, meaning that
// it is called with a remote server accepts our follow request.
func receive_AcceptFollow(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.receive_AcceptFollow"

	followingService := context.factory.Following()

	// Parse the Object.ID of the activity, which should be our original "Follow" activity
	userID, followingID, err := service.ParseProfileURL_AsFollowing(activity.Object().ID())

	if err != nil {
		return derp.Wrap(err, location, "Error parsing followingID", activity.Object().ID())
	}

	// Try to load the original "Following" record.
	// If it doesn't already exist, then this message is invalid.
	following := model.NewFollowing()
	if err := followingService.LoadByID(userID, followingID, &following); err != nil {
		return derp.Wrap(err, location, "Error loading following record", userID, followingID)
	}

	// RULE: Validate that the Following record matches the Accept
	if following.ProfileURL != activity.Actor().ID() {
		return derp.NewForbiddenError(location, "Invalid Accept", following.ProfileURL, activity.Actor().ID())
	}

	// Populate our "Following" record with the NAME and AVATAR of the remote Actor
	remoteActor, err := activity.Actor().Load()

	if err != nil {
		return derp.Wrap(err, location, "Error parsing remote actor", activity.Actor())
	}

	// Upgrade the "Following" record to ActivityPub
	following.Label = remoteActor.Name()
	following.IconURL = first.String(remoteActor.IconOrImage().URL(), following.IconURL)
	following.Method = model.FollowMethodActivityPub
	following.Secret = ""
	following.PollDuration = 30

	// Save the "Following" record to the database
	if err := followingService.SetStatusSuccess(&following); err != nil {
		return derp.Wrap(err, location, "Error saving following", following)
	}

	return nil
}
