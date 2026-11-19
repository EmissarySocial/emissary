package service

import (
	"iter"
	"slices"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/datetime"
	"github.com/benpate/hannibal/outbox"
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

	// RULE: Remove the Outbox2 activities too, because writers move to Outbox2 one at a time
	if err := service.outbox2Service.RemoveByObjectID(session, actorType, actorID, objectID); err != nil {
		return derp.Wrap(err, location, "Removing Outbox2 activities", objectID)
	}

	// Build the outgoing "Delete" activity. No top-level `id` (the Outbox mints one — a Delete has
	// no record of its own); `object.id` remains the URL of the thing being deleted. See D7.
	document := streams.NewDocument(mapof.Any{
		vocab.AtContext:     vocab.ContextTypeActivityStreams,
		vocab.PropertyActor: actor.ActorID(),
		vocab.PropertyType:  vocab.ActivityTypeDelete,
		vocab.PropertyTo:    vocab.NamespaceActivityStreamsPublic,
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

	// RULE: Remove the Outbox2 activity too, because writers move to Outbox2 one at a time
	if err := service.outbox2Service.removeByActivityURL(session, actorType, actorID, activityURL); err != nil {
		return derp.Wrap(err, location, "Removing Outbox2 activities", activityURL)
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
	// RULE: The document needs a client. A reply is also delivered to the author of the post
	// it answers, and that author is found by loading the post.
	document := streams.NewDocument(activity, streams.WithClient(service.deliveryClient(actorType, actorID)))
	recipientSeq := service.publishRecipients(session, actorType, actorID, document, config)

	// Filter and deliver to each recipient in turn
	if err := service.delivery().deliver(session, actorType, actorID, activity, permissions, recipientSeq); err != nil {
		return derp.Wrap(err, location, "Delivering activity")
	}

	// Success!!
	return nil
}

// deliveryClient returns the sending actor's client, which loads what an activity refers to.
// A service built without an ActivityStream service has none, and loads nothing.
func (service *Outbox) deliveryClient(actorType string, actorID primitive.ObjectID) streams.Client {

	if service.activityService == nil {
		return nil
	}

	return service.activityService.Client(actorType, actorID)
}

// delivery returns the follower delivery loop, wired with this service's dependencies
func (service *Outbox) delivery() followerDelivery {
	return followerDelivery{
		activityService: service.activityService,
		identityService: service.identityService,
		ruleService:     service.ruleService,
		domainEmail:     service.domainEmail,
		queue:           service.queue,
		host:            service.host,
	}
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
		addresseesAsFollowers(activity.RangeAddressees()),
		addresseesAsFollowers(activity.RangeInReplyTo()),
	)

	if config.hasRecipients {
		return joinIterators(
			addresseesAsFollowers(slices.Values(config.recipients)),
			addressees,
		)
	}

	return joinIterators(
		service.followerService.RangeFollowers(session, actorType, actorID),
		addressees,
	)
}

// permissionsToHex serializes a Permissions set to hex strings for queue-task storage
func permissionsToHex(permissions model.Permissions) []string {

	result := make([]string, 0, len(permissions))

	for _, permissionID := range permissions {
		result = append(result, permissionID.Hex())
	}

	return result
}
