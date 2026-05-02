package service

import (
	"net/url"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/sherlock"
	"github.com/davecgh/go-spew/spew"
	"github.com/labstack/gommon/random"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Follow guarantees that the user is following the specified URL.  If the user is already following this actor,
// then the Following record is returned. If not, a new Following record is created.
func (service *Following) Follow(session data.Session, userID primitive.ObjectID, actorID string) (model.Following, error) {

	const location = "service.Following.Follow"
	spew.Dump(location, "Attempting to follow URL", actorID)

	// If the actor ID is not already a valid URL, it's probably a username/handle,
	// so try to resolve it into a URL using Sherlock/WebFinger.
	if _, err := url.Parse(actorID); err != nil {

		// Look up the Actor from the Activity service
		actor, err := service.activityService.GetActor(actorID)

		if err != nil {
			return model.NewFollowing(), derp.Wrap(err, location, "Unable to find Actor for URL", actorID)
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
		return model.NewFollowing(), derp.Wrap(err, location, "Unable to load following for user and URL", userID, actorID)
	}

	// If the record is not found, then create a new one
	following.UserID = userID
	following.URL = actorID

	if err := service.Connect(session, &following); err != nil {
		return model.NewFollowing(), derp.Wrap(err, location, "Unable to connect to ActivityPub actor", following)
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
			return derp.Wrap(inner, location, "Unable to refresh ActivityPub Actor; Unable to mark `Following` record as `Failure`", err)
		}
		return derp.Wrap(err, location, "Unable to refresh ActivityPub Actor")
	}

	// Set values in the Following record...
	following.Label = actor.Name()
	following.ProfileURL = actor.ID()
	following.IconURL = actor.IconOrImage().URL()
	following.Username = actor.UsernameOrID()

	// Update the following status
	if err := service.SetStatusLoading(session, following); err != nil {
		return derp.Wrap(err, location, "Unable to set `Following` status to `Loading`", following)
	}

	// Prep arguments to send to queue consumers
	queueArgs := mapof.Any{
		"host":        service.host,
		"userId":      following.UserID.Hex(),
		"followingId": following.FollowingID.Hex(),
	}

	// Try to connect to push services (WebSub, ActivityPub, etc)
	// This runs in faster than usual because it affects the UX, but must
	// still write to the DB or else it may get skipped
	service.queue.NewTask("ConnectPushService", queueArgs)

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
		return derp.Wrap(err, location, "Error getting ActivityPub actor", following.UserID)
	}

	// Try to send the ActivityPub follow request
	followingURL := service.ActivityPubID(following)
	log.Debug().Str("loc", location).Msg("Sending ActivityPub Follow request to: " + remoteActor.ID())
	localActor.SendFollow(followingURL, remoteActor.ID())

	// Success!
	return nil
}

// ConnectWebSub attempts to connect to a remote user using WebSub (formerly PubSubHubbub).
func (service *Following) ConnectWebSub(following *model.Following, hub string) error {

	const location = "service.Following.connect_WebSub"

	var success string
	var failure string

	// Autocompute the topic.  Use "self" link first, or just the following URL
	self := following.GetLink("rel", model.LinkRelationSelf)

	// Update values in the following object
	following.Method = model.FollowingMethodWebSub
	following.Status = model.FollowingStatusSuccess
	following.URL = first.String(self.Href, following.URL)
	following.Secret = random.String(32)
	following.PollDuration = 30

	// Send request to the hub
	transaction := remote.Post(hub).
		Header("Accept", followingMimeStack).
		Form("hub.mode", "subscribe").
		Form("hub.topic", following.URL).
		Form("hub.callback", service.websubCallbackURL(following)).
		Form("hub.secret", following.Secret).
		Form("hub.lease_seconds", "2582000"). // 30 days
		Result(&success).
		Error(&failure)

	if err := transaction.Send(); err != nil {
		return derp.Wrap(err, location, "Unable to connect via WebSub subscription", hub, failure)
	}

	// Success!
	return nil
}
