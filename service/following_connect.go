package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/sherlock"
	"github.com/benpate/uri"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Follow guarantees that the user is following the specified URL.  If the user is already following this actor,
// then the Following record is returned. If not, a new Following record is created.
func (service *Following) Follow(session data.Session, userID primitive.ObjectID, actorID string) (model.Following, error) {

	const location = "service.Following.Follow"

	// If the actor ID is not a valid URL, it's probably a username/handle,
	// so try to resolve it into a URL using Sherlock/WebFinger.
	if uri.NotValidURL(actorID) {

		// Look up the Actor from the Activity service
		actor, err := service.activityService.GetActor(actorID)

		if err != nil {
			return model.NewFollowing(), derp.Wrap(err, location, "Finding Actor for URL", actorID)
		}

		actorID = actor.ID()
	}

	// Try to load the existing Following record for this user and URL
	following := model.NewFollowing()
	err := service.LoadByURL(session, userID, actorID, &following)

	if err == nil {
		return following, nil
	}

	if !derp.IsNotFound(err) {
		return model.NewFollowing(), derp.Wrap(err, location, "Loading following for user and URL", userID, actorID)
	}

	// If the record is not found, then create a new one
	following.UserID = userID
	following.URL = actorID

	if err := service.Connect(session, &following); err != nil {
		return model.NewFollowing(), derp.Wrap(err, location, "Connecting to ActivityPub actor", following)
	}

	// Success!
	return following, nil
}

// Connect attempts to connect to a new URL and determines how to follow it.
func (service *Following) Connect(session data.Session, following *model.Following) error {

	const location = "service.Following.Connect"

	// RULE: If we're already following via ActivityPub, then do not reconnect
	if following.Method == model.FollowingMethodActivityPub {
		return nil
	}

	// Try to load the Actor in the cache
	client := service.activityService.UserClient(following.UserID)
	actor, err := client.Load(following.URL, sherlock.AsActor())

	if err != nil {
		if inner := service.SetStatusFailure(session, following, "Unable to connect to ActivityPub Actor"); inner != nil {
			return derp.Wrap(inner, location, "Refreshing ActivityPub Actor; Unable to mark `Following` record as `Failure`", err)
		}
		return derp.Wrap(err, location, "Refreshing ActivityPub Actor")
	}

	// Set values in the Following record...
	following.Label = actor.Name()
	following.ProfileURL = actor.ID()
	following.IconURL = actor.IconOrImage().URL()
	following.Username = actor.UsernameOrID()

	// Update the following status
	if err := service.SetStatusLoading(session, following); err != nil {
		return derp.Wrap(err, location, "Setting `Following` status to `Loading`", following)
	}

	// Prep arguments to send to queue consumers
	queueArgs := mapof.Any{
		"hostname":    service.hostname,
		"userId":      following.UserID.Hex(),
		"followingId": following.FollowingID.Hex(),
	}

	// Try to connect to push services (now, only ActivityPub).  Published post-commit:
	// the task references this Following record, so it must not run until the enclosing
	// transaction has committed (otherwise the worker's session cannot see the row).
	postcommit.Publish(session, service.queue, "ConnectPushService", queueArgs)

	// Kool-Aid man says "ooooohhh yeah!"
	return nil
}

// ConnectActivityPub attempts to connect to a remote user using ActivityPub.
func (service *Following) ConnectActivityPub(session data.Session, following *model.Following, remoteActor *streams.Document) error {

	const location = "service.Following.ConnectActivityPub"

	// Update the Following record with the remote URL
	following.ProfileURL = remoteActor.ID()
	following.StatusMessage = "Pending ActivityPub connection"

	// Try to get the Actor (don't need Following channel)
	localActor, err := service.userService.ActivityPubActor(session, following.UserID)

	if err != nil {
		return derp.Wrap(err, location, "Getting ActivityPub actor", following.UserID)
	}

	// Send the ActivityPub Follow request as a post-commit queue task (F3): the signed HTTP
	// delivery happens after this transaction commits and is independently retryable. The signing
	// key is resolved in the sender consumer via SendLocator.Actor(localActor.ActorID()).
	followingURL := service.ActivityPubID(following)
	log.Debug().Str("loc", location).Msg("Sending ActivityPub Follow request to: " + remoteActor.ID())
	service.outboxService.SendFollow(session, localActor.ActorID(), followingURL, remoteActor.ID())

	// Success!
	return nil
}
