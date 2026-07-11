package service

import (
	"iter"
	"slices"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal"
	"github.com/benpate/hannibal/outbox"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/uri"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * Publish/Unpublish Methods
 ******************************************/

// Publish adds an OutboxMessage to the Actor's Outbox and sends notifications to all Followers.
// By default the activity fans out to every one of the Actor's followers plus the activity's own
// addressees; pass WithRecipients(...) to replace the follower fan-out with an explicit recipient
// list (e.g. author-only reaction delivery — see COLLECTIONS-REDESIGN.md D7b).
func (service *Outbox) Publish(session data.Session, actorType string, actorID primitive.ObjectID, activity streams.Document, permissions model.Permissions, options ...PublishOption) error {

	// TODO: This should become a background process.

	const location = "service.Outbox.Publish"
	if canTrace() {
		log.Trace().Str("location", location).Str("id", activity.ID()).Str("actor", actorID.Hex()).Str("object", activity.Object().ID()).Msg("Publishing object to outbox")
	}

	config := newPublishConfig(options...)

	// Generate an Actor for the Outbox
	actor, err := service.getActor(session, actorType, actorID)

	if err != nil {
		return derp.Wrap(err, location, "Loading Actor", actorType, actorID)
	}

	// Write a new OutboxMessage to the database
	outboxMessage := model.NewOutboxMessage()
	outboxMessage.ActorType = actorType
	outboxMessage.ActorID = actorID
	outboxMessage.ActorURL = actor.ActorID()
	outboxMessage.ObjectID = activity.Object().ID()
	outboxMessage.ActivityType = activity.Type()
	outboxMessage.Permissions = permissions

	// If the activity ALREADY carries its own canonical ID (it is a first-class
	// activity with its own record, e.g. a Like/Dislike/Announce or a Block), keep
	// that ID and store it on the OutboxMessage so the message is findable by it.
	// Otherwise (the published thing is an OBJECT, or an Undo/Delete event with no
	// record of its own) the Outbox mints a fresh ID for it. See COLLECTIONS-REDESIGN.md D7.
	if activityID := activity.ID(); activityID != "" {
		outboxMessage.ActivityURL = activityID
	}

	if err := service.Save(session, &outboxMessage, "Publishing"); err != nil {
		return derp.Wrap(err, location, "Unable to save outbox message", outboxMessage)
	}

	// Build the recipient set (followers + addressees by default; an explicit recipient list
	// replaces the follower fan-out when WithRecipients was passed — see D7b).
	recipients := service.publishRecipients(session, actorType, actorID, activity, config)

	// Only mint an ID for activities that arrived without one; never overwrite a
	// canonical activity ID (D7). outboxMessage.ActivityPubURL() returns the stored
	// ActivityURL when present, else the minted <actor>/pub/outbox/<id> URL.
	activityMap := activity.Map()
	activityMap[vocab.PropertyID] = outboxMessage.ActivityPubURL()
	ruleFilter := service.ruleService.Filter(actorID, WithBlocksOnly())

	isLocalhost := uri.IsLocalHostname(service.host)

	for follower := range recipients {

		// Resolve the recipient's host from InboxURL, falling back to ProfileURL. Addressees
		// (e.g. a reaction's target author, added via addresseesAsFollowers) are constructed with
		// ONLY a ProfileURL, so testing InboxURL alone would evaluate IsLocalHostname("") == false
		// and wrongly drop them on a localhost/dev domain — silently breaking author-only delivery
		// and the local loopback the reaction projection depends on. See COLLECTIONS-REDESIGN.md D8.
		recipientHost := follower.Actor.InboxURL
		if recipientHost == "" {
			recipientHost = follower.Actor.ProfileURL
		}

		// RULE: Only deliver to Followers on the same network as the Actor
		// (local can send to local, public can send to public, but local cannot send to public)
		if uri.IsLocalHostname(recipientHost) != isLocalhost {
			continue
		}

		// RULE: Do not send to blocked Followers
		if !ruleFilter.AllowSend(session, follower.Actor.ProfileURL) {
			log.Trace().Msg("Follower blocked by rule filter: " + follower.Actor.ProfileURL)
			continue
		}

		// RULE: Do not send to Followers who do not have permissions to view this activity
		if !service.identityService.HasPermissions(session, follower.Method, follower.Actor.ProfileURL, permissions) {
			log.Trace().Msg("Follower does not have permissions to view this activity: " + follower.Actor.ProfileURL)
			continue
		}

		log.Debug().Msg("Sending notification to Follower: " + follower.Actor.ProfileURL)

		switch follower.Method {

		case model.FollowerMethodActivityPub:
			service.sendNotification_ActivityPub(&actor, &follower, activityMap)

		case model.FollowerMethodEmail:
			service.sendNotification_Email(&follower, activityMap)

		default:
			derp.Report(derp.Internal(location, "Unknown Follower Method.  This should never happen", follower))
		}
	}

	// Success!!
	return nil
}

// DeleteActivity removes the OutboxMessages that published an OBJECT (keyed by the object's URL)
// and sends a "Delete" activity wrapping a Tombstone of that object to followers. A Delete is an
// event with no activity record of its own, so the outgoing Delete carries no top-level `id` (the
// Outbox mints one); only the wrapped `object.id` — the thing being deleted — is meaningful.
func (service *Outbox) DeleteActivity(session data.Session, actorType string, actorID primitive.ObjectID, objectID string, permissions model.Permissions) error {

	const location = "service.Outbox.DeleteActivity"

	actor, err := service.getActor(session, actorType, actorID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to get Actor", actorType, actorID)
	}

	// Remove the outbox messages that published this OBJECT (matched by objectId).
	if err := service.removeOutboxMessagesByObjectID(session, actorType, actorID, objectID); err != nil {
		return derp.Wrap(err, location, "Unable to remove outbox messages", objectID)
	}

	// Build the outgoing "Delete" activity. No top-level `id` (the Outbox mints one — a Delete has
	// no record of its own); `object.id` remains the URL of the thing being deleted. See D7.
	document := streams.NewDocument(mapof.Any{
		vocab.AtContext:     vocab.ContextTypeActivityStreams,
		vocab.PropertyActor: actor.ActorID(),
		vocab.PropertyType:  vocab.ActivityTypeDelete,
		vocab.PropertyTo:    vocab.NamespacePublic,
		vocab.PropertyObject: mapof.Any{
			vocab.PropertyID:   objectID,
			vocab.PropertyType: vocab.ObjectTypeTombstone,
		},
		vocab.PropertyPublished: hannibal.TimeFormat(time.Now()),
	})

	if err := service.Publish(session, actorType, actorID, document, permissions); err != nil {
		return derp.Wrap(err, location, "Unable to publish DELETE activity", objectID)
	}

	return nil
}

// UndoActivity removes the OutboxMessage that published a first-class ACTIVITY (keyed by that
// activity's own canonical URL) and sends an "Undo" activity that EMBEDS the original activity
// inline. The original activity is embedded (not referenced by URL) because the caller has often
// already hard-deleted the underlying record — a receiver, including our own loopback, must be
// able to un-project without dereferencing a URL that now 404s. See COLLECTIONS-REDESIGN.md D7.
func (service *Outbox) UndoActivity(session data.Session, actorType string, actorID primitive.ObjectID, originalActivity mapof.Any, permissions model.Permissions, options ...PublishOption) error {

	const location = "service.Outbox.UndoActivity"

	actor, err := service.getActor(session, actorType, actorID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to get Actor", actorType, actorID)
	}

	// The activity being undone is identified by its own canonical URL.
	activityURL := originalActivity.GetString(vocab.PropertyID)

	// Remove the outbox message(s) that published the original ACTIVITY (matched by activityUrl).
	if err := service.removeOutboxMessagesByActivityURL(session, actorType, actorID, activityURL); err != nil {
		return derp.Wrap(err, location, "Unable to remove outbox messages", activityURL)
	}

	// Build the outgoing "Undo" activity with the original activity embedded inline. No top-level
	// `id` (the Outbox mints one — an Undo has no record of its own).
	undo := mapof.Any{
		vocab.AtContext:         vocab.ContextTypeActivityStreams,
		vocab.PropertyActor:     actor.ActorID(),
		vocab.PropertyType:      vocab.ActivityTypeUndo,
		vocab.PropertyObject:    originalActivity,
		vocab.PropertyPublished: hannibal.TimeFormat(time.Now()),
	}

	// Mirror the original activity's top-level audience (`to`/`cc`) onto the Undo. Publish derives
	// its default recipients from the DOCUMENT's own addressees (RangeAddressees does not recurse
	// into the embedded object), so without this the Undo of a broadcast Announce — whose addressing
	// lives on the embedded object — would reach only followers and never loop back to the author to
	// un-project. (Author-only Like/Dislike Undos carry no to/cc and rely on WithRecipients instead.)
	if to, exists := originalActivity[vocab.PropertyTo]; exists {
		undo[vocab.PropertyTo] = to
	}
	if cc, exists := originalActivity[vocab.PropertyCC]; exists {
		undo[vocab.PropertyCC] = cc
	}

	document := streams.NewDocument(undo)

	// Forward the caller's PublishOptions (spread!) so an Undo can honor the same author-only
	// delivery as the original activity. A dropped spread here would fan the Undo out to all
	// followers even though the original reaction was author-only. See COLLECTIONS-REDESIGN.md D7b.
	if err := service.Publish(session, actorType, actorID, document, permissions, options...); err != nil {
		return derp.Wrap(err, location, "Unable to publish UNDO activity", activityURL)
	}

	return nil
}

// removeOutboxMessagesByObjectID deletes every OutboxMessage in this Actor's outbox that
// published the given OBJECT (matched by the object's URL). Used by DeleteActivity.
func (service *Outbox) removeOutboxMessagesByObjectID(session data.Session, actorType string, actorID primitive.ObjectID, objectID string) error {

	const location = "service.Outbox.removeOutboxMessagesByObjectID"

	messages, err := service.RangeByObjectID(session, actorType, actorID, objectID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to load outbox messages", objectID)
	}

	for message := range messages {
		if err := service.Delete(session, &message, "Un-Publishing"); err != nil {
			return derp.Wrap(err, location, "Unable to delete outbox message", message)
		}
	}

	return nil
}

// removeOutboxMessagesByActivityURL deletes every OutboxMessage in this Actor's outbox that
// published the given first-class ACTIVITY (matched by the activity's own URL). Used by UndoActivity.
func (service *Outbox) removeOutboxMessagesByActivityURL(session data.Session, actorType string, actorID primitive.ObjectID, activityURL string) error {

	const location = "service.Outbox.removeOutboxMessagesByActivityURL"

	messages, err := service.RangeByActivityURL(session, actorType, actorID, activityURL)

	if err != nil {
		return derp.Wrap(err, location, "Unable to load outbox messages", activityURL)
	}

	for message := range messages {
		if err := service.Delete(session, &message, "Un-Publishing"); err != nil {
			return derp.Wrap(err, location, "Unable to delete outbox message", message)
		}
	}

	return nil
}

func (service *Outbox) getActor(session data.Session, actorType string, actorID primitive.ObjectID) (outbox.Actor, error) {

	switch actorType {

	case model.FollowerTypeUser:
		return service.userService.ActivityPubActor(session, actorID)

	case model.FollowerTypeStream:
		return service.streamService.ActivityPubActor(session, actorID)

	case model.FollowerTypeApplication:

	case model.FollowerTypeSearch:

	case model.FollowerTypeSearchDomain:

	}

	return outbox.Actor{}, derp.Internal("service.Outbox.getActor", "Unknown Actor Type", actorType)
}

/******************************************
 * Notification Protocols
 ******************************************/

// publishRecipients returns the set of Followers to notify for a Publish. By default this is the
// Actor's followers plus the activity's own addressees. When WithRecipients(...) was passed, the
// explicit recipient list REPLACES the follower fan-out (author-only delivery — D7b); the
// activity's own addressees are still included.
func (service *Outbox) publishRecipients(session data.Session, actorType string, actorID primitive.ObjectID, activity streams.Document, config PublishConfig) iter.Seq[model.Follower] {

	addressees := joinIterators(
		service.addresseesAsFollowers(activity.RangeAddressees()),
		service.addresseesAsFollowers(activity.RangeInReplyTo()),
	)

	if config.hasRecipients {
		return joinIterators(
			service.addresseesAsFollowers(slices.Values(config.recipients)),
			addressees,
		)
	}

	return joinIterators(
		service.followerService.RangeFollowers(session, actorType, actorID),
		addressees,
	)
}

func (service *Outbox) addresseesAsFollowers(addressees iter.Seq[string]) iter.Seq[model.Follower] {

	return func(yield func(model.Follower) bool) {

		uniquer := streams.NewUniquer[string]()

		for addressee := range uniquer.Range(addressees) {
			follower := model.NewFollower()
			follower.Actor.ProfileURL = addressee
			follower.Method = model.FollowerMethodActivityPub
			follower.StateID = model.FollowerStateActive

			if !yield(follower) {
				return
			}
		}
	}
}

// sendNotifications_ActivityPub sends ActivityPub updates to all Followers
// TODO: HIGH: This should be a background task with retries
func (service *Outbox) sendNotification_ActivityPub(actor *outbox.Actor, follower *model.Follower, activity mapof.Any) {
	if err := actor.SendOne(follower.Actor.ProfileURL, activity); err != nil {
		derp.Report(derp.Wrap(err, "service.Outbox.sendNotifications_ActivityPub", "Unable to send ActivityPub notification", follower.Actor.ProfileURL))
	}
}

// sendNotifications_Email sends email notifications to all "email" Followers
func (service *Outbox) sendNotification_Email(follower *model.Follower, activity mapof.Any) {

	const location = "service.Outbox.sendNotifications_Email"

	if err := service.domainEmail.SendFollowerActivity(follower, activity); err != nil {
		derp.Report(derp.Wrap(err, location, "Unable to send email", follower))
	}
}
