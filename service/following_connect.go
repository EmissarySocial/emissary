package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/sherlock"
	"github.com/benpate/turbine/queue"
	"github.com/labstack/gommon/random"
	"github.com/rs/zerolog/log"
)

// Connect attempts to connect to a new URL and determines how to follow it.
func (service *Following) Connect(session data.Session, following *model.Following) error {

	const location = "service.Following.Connect"

	// RULE: If we're already following via ActivityPub, then do not reconnect
	if following.Method == model.FollowingMethodActivityPub {
		return nil
	}

	// Try to load the Actor in the cache (allow cached values)
	client := service.activityService.UserClient(following.UserID)
	actor, err := client.Load(following.URL, sherlock.AsActor(), ascache.WithWriteOnly())

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

	// Try to load an initial list of messages from the actor's outbox
	// This runs in faster than usual because it affects the UX, but must
	// still write to the DB or else it may get skipped
	service.queue.NewTask("PollFollowing-Record", queueArgs, queue.WithPriority(32))

	// Try to connect to push services (WebSub, ActivityPub, etc)
	// This runs in faster than usual because it affects the UX, but must
	// still write to the DB or else it may get skipped
	service.queue.NewTask("ConnectPushService", queueArgs, queue.WithPriority(32))

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
