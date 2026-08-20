package activitypub_user

import (
	"time"

	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/rs/zerolog/log"
)

// init registers the handler for inbound Like activities
func init() {
	inboxRouter.Add(vocab.ActivityTypeLike, vocab.Any, inbox_LikeOrAnnounce)
}

// inbox_LikeOrAnnounce handles all Like/Dislike/Announce activities delivered to a User's inbox.
func inbox_LikeOrAnnounce(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.inbox_LikeOrAnnounce"

	// RULE: If the Activity does not have an ID, then make a new "fake" one.
	if activity.ID() == "" {
		activity.SetProperty(vocab.PropertyID, activitypub.FakeActivityID(activity))
	}

	// Collect the ActorID for this Activity
	actorID := activity.ActorID()

	if actorID == "" {
		return derp.BadRequest(location, "Activity must have an ActorID", activity.Value())
	}

	// ACCEPTANCE: A User's inbox is a curated newsfeed, so it does not accept arbitrary traffic.
	// Accept only when the activity is a private message to us, a self-message, or comes from an
	// actor we Follow (see COLLECTIONS-REDESIGN.md D8). `following` is nil for the first two cases.
	following, accepted := isMessageAllowed(context, activity, actorID)

	if !accepted {
		// SILENT DROP: return success (not an error — a 5xx makes the sender retry a delivery we
		// will never accept), plus a console line for dev visibility (resolved Q2).
		log.Debug().Str("location", location).Str("actor", actorID).Str("activity", activity.ID()).Msg("Dropping inbox activity from a non-followed, non-self actor")
		return nil
	}

	// RULE: Evaluate the sender against this User's rules (R9). Self-messages are exempt -- a local
	// user's own reactions loop back through this same funnel and must never be filtered. A remote
	// blocked sender is already rejected at the Stage-2 inbox gate; the BLOCK branch here is the
	// authoritative gate for the self-loopback path and defense-in-depth for the rest.
	var disposition model.RuleDisposition

	if actorID != context.user.ActivityPubURL() {
		d, err := context.factory.Rule().ActorDisposition(context.session, context.user.UserID, activity, time.Now().Unix())

		if err != nil {
			return derp.Wrap(err, location, "Checking rules", actorID)
		}

		disposition = d
	}

	// BLOCK drops the reaction entirely: no fetch (possibly from a blocked server), no cache write,
	// no response-collection entry, no newsfeed item.
	if disposition.IsBlocked() {
		return nil
	}

	// Load the original ActivityStream document being Liked/Announced (which also adds it to the cache)
	document, err := activity.Object().Load()

	if err != nil {
		return derp.Wrap(err, location, "Loading ActivityStream document", activity.Object().ID())
	}

	// Add the activity into the ActivityStream cache (for statistics)
	if err := context.factory.ActivityStream().Save(activity); err != nil {
		return derp.Wrap(err, location, "Saving activity", activity.ID())
	}

	// Project this Like/Dislike/Announce into the reacted-to object's response collection (if the
	// object is a local Stream). This inbound funnel is the SOLE writer of response CollectionItems
	// (D6); it is a no-op for remote/non-Stream objects.
	if err := context.factory.Stream().AddResponseCollectionItem(context.session, document.ID(), activity.Type(), activity.ID()); err != nil {
		return derp.Wrap(err, location, "Projecting response into collection", activity.ID())
	}

	// Add the reacted-to message into the User's newsfeed — ONLY when it arrived from an actor we
	// Follow (self-messages and private-messages without a Following record have no newsfeed side),
	// and NOT when that actor is muted: a muted actor's reactions stay in the aggregate collection
	// totals (written above, R9) but never surface in the feed.
	if (following != nil) && !disposition.IsMuted() {
		originType := getOriginType(activity.Type())

		if err := context.factory.Following().SaveNewsItem(context.session, following, document, originType); err != nil {
			return derp.Wrap(err, location, "Saving news item", context.user.UserID, activity.Value())
		}
	}

	// Success.
	return nil
}

// isMessageAllowed decides whether an inbound activity is allowed into this User's inbox, per D8.
// It returns the matching Following record (non-nil ONLY when acceptance was granted BY a Following
// relationship — used to drive the newsfeed side-effect) and a boolean acceptance verdict.
func isMessageAllowed(context Context, activity streams.Document, actorID string) (*model.Following, bool) {

	const location = "handler.activitypub_user.isMessageAllowed"

	// RULE: Allow messages if the Sender and Receiver are identical (self-message).
	if actorID == context.user.ActivityPubURL() {
		return nil, true
	}

	// RULE: Allow Private message addressed specifically to this user.
	if activity.NotPublic() {
		return nil, true
	}

	// RULE: Allow if the sender is in our Following collection.
	followingService := context.factory.Following()
	following := model.NewFollowing()

	if err := followingService.LoadByURL(context.session, context.user.UserID, actorID, &following); err == nil {
		return &following, true

	} else if !derp.IsNotFound(err) {
		// A real load error (not merely "no record") — report but do not accept.
		derp.Report(derp.Wrap(err, location, "Querying Following record", actorID))
	}

	// Otherwise: not accepted.
	return nil, false
}
