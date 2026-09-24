package service

import (
	"iter"
	"maps"
	"slices"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/datetime"
	"github.com/benpate/hannibal/outbox"
	"github.com/benpate/hannibal/sender"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/uri"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// taskOutboxPublish is the queue-task name for the post-commit follower fan-out
const taskOutboxPublish = "Outbox-Publish"

/******************************************
 * Publish/Unpublish Methods
 ******************************************/

// DeleteActivity removes the OutboxMessages that published an object, and sends
// a "Delete" activity wrapping a Tombstone of that object to followers.
func (service *Outbox) DeleteActivity(session data.Session, actorType string, actorID primitive.ObjectID, objectID string, permissions model.Permissions) error {

	const location = "service.Outbox.DeleteActivity"

	// Load the Actor that is deleting the object
	actor, err := service.getActor(session, actorType, actorID)

	if err != nil {
		return derp.Wrap(err, location, "Getting Actor", actorType, actorID)
	}

	// Remove the outbox messages that published this OBJECT (matched by objectId).
	if err := service.removeOutboxMessagesByObjectID(session, actorType, actorID, objectID); err != nil {
		return derp.Wrap(err, location, "Removing outbox messages", objectID)
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
		vocab.PropertyPublished: datetime.Now(),
	})

	// Publish the "Delete" activity to followers
	if err := service.Publish(session, actorType, actorID, document, permissions); err != nil {
		return derp.Wrap(err, location, "Publishing DELETE activity", objectID)
	}

	// Gone, but not forgotten
	return nil
}

// UndoActivity removes the OutboxMessage that published a first-class activity, and
// sends an "Undo" activity that embeds the original activity inline.
func (service *Outbox) UndoActivity(session data.Session, actorType string, actorID primitive.ObjectID, originalActivity mapof.Any, permissions model.Permissions, options ...PublishOption) error {

	const location = "service.Outbox.UndoActivity"

	// Load the Actor that is undoing the activity
	actor, err := service.getActor(session, actorType, actorID)

	if err != nil {
		return derp.Wrap(err, location, "Getting Actor", actorType, actorID)
	}

	// The activity being undone is identified by its own canonical URL.
	activityURL := originalActivity.GetString(vocab.PropertyID)

	// Remove the outbox message(s) that published the original ACTIVITY (matched by activityUrl).
	if err := service.removeOutboxMessagesByActivityURL(session, actorType, actorID, activityURL); err != nil {
		return derp.Wrap(err, location, "Removing outbox messages", activityURL)
	}

	// Build the outgoing "Undo" activity with the original activity embedded inline, so a receiver
	// can un-project it without dereferencing a URL that may now 404. No top-level `id` (D7).
	undo := mapof.Any{
		vocab.AtContext:         vocab.ContextTypeActivityStreams,
		vocab.PropertyActor:     actor.ActorID(),
		vocab.PropertyType:      vocab.ActivityTypeUndo,
		vocab.PropertyObject:    originalActivity,
		vocab.PropertyPublished: datetime.Now(),
	}

	// Mirror the original's top-level `to`/`cc` onto the Undo, because RangeAddressees does not
	// recurse into the embedded object. Without this, the Undo of an Announce never reaches its author.
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
		return derp.Wrap(err, location, "Publishing UNDO activity", activityURL)
	}

	// Ctrl+Z
	return nil
}

// Deliver runs the post-commit follower fan-out for an already-published activity, queuing
// one ActivityPub delivery task per recipient and sending email notifications inline.
func (service *Outbox) Deliver(session data.Session, actorType string, actorID primitive.ObjectID, activity mapof.Any, permissions model.Permissions, recipients []string, hasRecipients bool) error {

	const location = "service.Outbox.Deliver"

	// Rebuild the PublishConfig exactly as Publish had it. An explicit (even empty) recipient
	// list suppresses the follower fan-out — WithRecipients sets both fields (D7b).
	config := PublishConfig{}
	if hasRecipients {
		config = newPublishConfig(WithRecipients(recipients...))
	}

	// Rebuild the recipient set (followers + addressees by default; the override list otherwise),
	// iterating lazily from the activity's FULL addressing. publishRecipients reads to/cc AND
	// bto/bcc (RangeAddressees), so enumeration must see the blind fields before they are stripped.
	document := streams.NewDocument(activity)
	recipientSeq := service.publishRecipients(session, actorType, actorID, document, config)

	// RULE: Deliver the stripped payload, never the original. Blind recipients still RECEIVE the
	// activity (enumerated above from the full addressing) but must not SEE the blind-address list.
	payload := maps.Clone(activity)
	delete(payload, vocab.PropertyBTo)
	delete(payload, vocab.PropertyBCC)

	// The authoritative signer URL (stamped by Publish) — the delivery tasks resolve the key from it.
	actorURL := payload.GetString(vocab.PropertyActor)

	// The rules userID for the egress gate: a User actor binds their own rules; Stream and
	// SearchQuery actorIDs are NOT UserIDs, so those actors bind the admin tier alone.
	rulesUserID := ruleUserID(actorType, actorID)
	isLocalhost := uri.IsLocalHostname(service.host)

	// RULE: ask once, not once per Follower.  DomainEmail.Send now treats an unconfigured SMTP
	// connection as an error, and this loop can hold thousands of Followers -- reporting one
	// error each would flood the log on a domain that deliberately runs without email.
	canSendEmail := service.domainEmail.IsConfigured()

	if !canSendEmail {
		log.Debug().Str("location", location).Msg("SMTP is not configured. Skipping email Followers.")
	}

	// Filter and deliver to each recipient in turn
	for follower := range recipientSeq {

		// Resolve the recipient's host from InboxURL, falling back to ProfileURL, because
		// addressees carry only a ProfileURL and would otherwise be dropped on a localhost domain (D8).
		recipientHost := follower.Actor.InboxURL
		if recipientHost == "" {
			recipientHost = follower.Actor.ProfileURL
		}

		// RULE: Only deliver to Followers on the same network as the Actor
		// (local can send to local, public can send to public, but local cannot send to public)
		if uri.IsLocalHostname(recipientHost) != isLocalhost {
			continue
		}

		// RULE: Delivery to blocked recipients is halted (R4) -- BOTH methods, email included
		if service.ruleService.DeliveryBlocked(session, rulesUserID, follower.Actor.ProfileURL) {
			log.Trace().Msg("Follower blocked by rule: " + follower.Actor.ProfileURL)
			continue
		}

		// RULE: Do not send to Followers who do not have permissions to view this activity
		if !service.identityService.HasPermissions(session, follower.Method, follower.Actor.ProfileURL, permissions) {
			log.Trace().Msg("Follower does not have permissions to view this activity: " + follower.Actor.ProfileURL)
			continue
		}

		log.Debug().Msg("Sending notification to Follower: " + follower.Actor.ProfileURL)

		// Deliver by the Follower's chosen method
		switch follower.Method {

		case model.FollowerMethodActivityPub:
			service.deliverActivityPub(session, actorURL, &follower, payload)

		case model.FollowerMethodEmail:
			if canSendEmail {
				service.sendNotification_Email(&follower, payload)
			}

		default:
			derp.Report(derp.Internal(location, "Unknown Follower Method.  This should never happen", follower))
		}
	}

	// Success!!
	return nil
}

// Publish saves an OutboxMessage for the activity, then queues a post-commit "Outbox-Publish"
// task that delivers it to the Actor's followers and the activity's own addressees.
func (service *Outbox) Publish(session data.Session, actorType string, actorID primitive.ObjectID, activity streams.Document, permissions model.Permissions, options ...PublishOption) error {

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

	// Keep an activity's own canonical ID (a Like, Announce, or Block has a record of its own),
	// so the message is findable by it. Objects and Undo/Delete events get a minted ID (D7).
	if activityID := activity.ID(); activityID != "" {
		outboxMessage.ActivityURL = activityID
	}

	if err := service.Save(session, &outboxMessage, "Publishing"); err != nil {
		return derp.Wrap(err, location, "Saving outbox message", outboxMessage)
	}

	// Stamp the canonical `id` (the stored ActivityURL, else a minted outbox URL) and the actor
	// URL, so the delivery tasks can resolve the signing key through SendLocator.Actor.
	activityMap := activity.Map()
	activityMap[vocab.PropertyID] = outboxMessage.ActivityPubURL()
	activityMap[vocab.PropertyActor] = actor.ActorID()

	// Hand the fan-out to a post-commit task, which Outbox.Deliver runs after this transaction
	// commits. Permissions travel as hex strings, which survive task storage round-trips.
	postcommit.Publish(session, service.queue, taskOutboxPublish, mapof.Any{
		"hostname":      uri.Hostname(service.host),
		"actorType":     actorType,
		"actorId":       actorID.Hex(),
		"activity":      activityMap,
		"permissions":   permissionsToHex(permissions),
		"recipients":    config.recipients,
		"hasRecipients": config.hasRecipients,
	})

	// Success!!
	return nil
}

// getActor returns the outbox Actor for the named actor type and ID
func (service *Outbox) getActor(session data.Session, actorType string, actorID primitive.ObjectID) (outbox.Actor, error) {

	const location = "service.Outbox.getActor"

	switch actorType {

	case model.FollowerTypeUser:
		return service.userService.ActivityPubActor(session, actorID)

	case model.FollowerTypeStream:
		return service.streamService.ActivityPubActor(session, actorID)

	}

	// Application, Search, and SearchDomain are real Actors, but none of them has a route that
	// serves an Outbox item, so a message they cannot identify must never reach the Outbox.
	return outbox.Actor{}, derp.Internal(location, "Actor type cannot own an Outbox message", actorType)
}

// removeOutboxMessagesByObjectID deletes every OutboxMessage in this Actor's outbox that
// published the given object, matched by the object's URL.
func (service *Outbox) removeOutboxMessagesByObjectID(session data.Session, actorType string, actorID primitive.ObjectID, objectID string) error {

	const location = "service.Outbox.removeOutboxMessagesByObjectID"

	messages, err := service.RangeByObjectID(session, actorType, actorID, objectID)

	if err != nil {
		return derp.Wrap(err, location, "Loading outbox messages", objectID)
	}

	for message := range messages {
		if err := service.Delete(session, &message, "Un-Publishing"); err != nil {
			return derp.Wrap(err, location, "Deleting outbox message", message)
		}
	}

	return nil
}

// removeOutboxMessagesByActivityURL deletes every OutboxMessage in this Actor's outbox that
// published the given first-class activity, matched by the activity's own URL.
func (service *Outbox) removeOutboxMessagesByActivityURL(session data.Session, actorType string, actorID primitive.ObjectID, activityURL string) error {

	const location = "service.Outbox.removeOutboxMessagesByActivityURL"

	messages, err := service.RangeByActivityURL(session, actorType, actorID, activityURL)

	if err != nil {
		return derp.Wrap(err, location, "Loading outbox messages", activityURL)
	}

	for message := range messages {
		if err := service.Delete(session, &message, "Un-Publishing"); err != nil {
			return derp.Wrap(err, location, "Deleting outbox message", message)
		}
	}

	return nil
}

/******************************************
 * Notification Protocols
 ******************************************/

// publishRecipients returns the Followers to notify for a Publish: the Actor's followers, or the
// WithRecipients list when one was given, plus the activity's own addressees in either case.
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

// deliverActivityPub queues one retryable OutboxSendToSingleRecipient task that delivers the
// activity to a single ActivityPub follower.
func (service *Outbox) deliverActivityPub(session data.Session, actorURL string, follower *model.Follower, activity mapof.Any) {

	const location = "service.Outbox.deliverActivityPub"

	inboxURL := follower.Actor.InboxURL

	if inboxURL == "" {
		inboxURL = service.resolveInboxURL(follower.Actor.ProfileURL)
	}

	// A follower with no resolvable inbox cannot be delivered to. This matches the former
	// behavior (actor.SendOne failed and reported) — the delivery is dropped, not retried.
	if inboxURL == "" {
		derp.Report(derp.Internal(location, "Unable to resolve inbox URL for follower", follower.Actor.ProfileURL))
		return
	}

	postcommit.Publish(session, service.queue, sender.OutboxSendToSingleRecipient, mapof.Any{
		"actor":    actorURL,
		"inbox":    inboxURL,
		"activity": activity,
	})
}

// sendNotification_Email sends an email notification of the activity to a single "email" Follower
func (service *Outbox) sendNotification_Email(follower *model.Follower, activity mapof.Any) {

	const location = "service.Outbox.sendNotifications_Email"

	if err := service.domainEmail.SendFollowerActivity(follower, activity); err != nil {
		derp.Report(derp.Wrap(err, location, "Sending email", follower))
	}
}

// permissionsToHex serializes a Permissions set to hex strings for queue-task storage
func permissionsToHex(permissions model.Permissions) []string {

	result := make([]string, 0, len(permissions))

	for _, permissionID := range permissions {
		result = append(result, permissionID.Hex())
	}

	return result
}

// addresseesAsFollowers presents a list of addressee URLs as synthetic ActivityPub Followers, so that
// directly-addressed recipients can share the follower fan-out path
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

// resolveInboxURL loads an Actor from its profile URL and returns its preferred inbox URL,
// or an empty string when the Actor cannot be loaded.
func (service *Outbox) resolveInboxURL(profileURL string) string {

	const location = "service.Outbox.resolveInboxURL"

	if profileURL == "" {
		return ""
	}

	actor, err := service.activityService.AppClient().Load(profileURL)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Loading actor for inbox URL", profileURL))
		return ""
	}

	if actor.NotActor() {
		return ""
	}

	return actor.PreferredInbox()
}
