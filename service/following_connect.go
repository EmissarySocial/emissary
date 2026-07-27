package service

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/asrules"
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

		// RULE: a Following paused by a (since-deleted) block resumes only on an explicit
		// re-follow like this one -- never automatically (R8)
		if following.Status == model.FollowingStatusPaused {
			if err := service.Resume(session, &following); err != nil {
				return model.NewFollowing(), derp.Wrap(err, location, "Resuming paused Following", following.FollowingID)
			}
		}

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

	// RULE: R11 -- a Following cannot be created while a block still covers this actor. This mirrors
	// the identical check in Resume, and it is what MAKES the reveal below safe: the refusal is now an
	// explicit, friendly decision about BLOCK alone, instead of an opaque 403 from the rules client
	// that also swept up mutes and labels.
	disposition, err := service.ruleService.DispositionForKeys(session, following.UserID, model.ActorMatchKeys(following.URL), time.Now().Unix())

	if err != nil {
		return derp.Wrap(err, location, "Checking rules before connecting", following.URL)
	}

	if disposition.IsBlocked() {
		return derp.Validation("You have blocked this account. Remove the block rule before following it.")
	}

	// Try to load the Actor in the cache.
	//
	// RULE: an explicit follow is a deliberate request FOR this actor, not unsolicited content, so the
	// viewer's own rules must not refuse the fetch. R19 exists to keep hidden content from being pulled
	// in behind the User's back; it should never stop the User from acting on their own intent. Without
	// the reveal, merely MUTING an account made it impossible to follow -- and the failure surfaced as
	// a raw 403 that took down the whole enclosing form. BLOCK is still refused, above.
	client := service.activityService.UserClient(following.UserID)
	actor, err := client.Load(following.URL, sherlock.AsActor(), asrules.WithReveal(true))

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

	// Persist, folding onto any existing follow of this actor so a duplicate (double-click,
	// concurrent request, or a create that raced another) updates the original row instead of
	// inserting a second one. profileUrl is resolved above, so reconciliation is now meaningful;
	// idx_Following_User_Profile_Unique is the backstop that turns a lost race into a retry.
	if err := service.reconcileAndSave(session, following, func() error {
		return service.SetStatusLoading(session, following)
	}); err != nil {
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

// Resume re-activates a PAUSED Following on the user's explicit re-follow (R8): ActivityPub rows
// re-send their Follow request (an Undo/Follow went out when the row was paused), poll rows simply
// resume polling. Never called by the restore pass -- re-following is a user decision.
func (service *Following) Resume(session data.Session, following *model.Following) error {

	const location = "service.Following.Resume"

	// RULE: only PAUSED rows can be resumed
	if following.Status != model.FollowingStatusPaused {
		return nil
	}

	// RULE: R11 -- a Following cannot resume while a block still covers this actor
	keys := append(model.ActorMatchKeys(following.URL), model.ActorMatchKeys(following.ProfileURL)...)
	disposition, err := service.ruleService.DispositionForKeys(session, following.UserID, keys, time.Now().Unix())

	if err != nil {
		return derp.Wrap(err, location, "Checking rules before resuming", following.URL)
	}

	if disposition.IsBlocked() {
		return derp.Validation("You have blocked this account. Remove the block rule before following it.")
	}

	// Re-send the ActivityPub Follow request (poll rows have nothing to send)
	if following.Method == model.FollowingMethodActivityPub {

		localActor, err := service.userService.ActivityPubActor(session, following.UserID)

		if err != nil {
			return derp.Wrap(err, location, "Getting ActivityPub actor", following.UserID)
		}

		service.outboxService.SendFollow(session, localActor.ActorID(), service.ActivityPubID(following), following.ProfileURL)
	}

	// Back to LOADING: the Accept handler (or the next poll) promotes it to SUCCESS from here
	return service.SetStatusLoading(session, following)
}
