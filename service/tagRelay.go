package service

import (
	"slices"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/collections"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * Tag Relays
 *
 * The @application actor follows one tags.pub actor per
 * #hashtag that someone here follows, so tagged posts from
 * across the fediverse arrive once for the whole server.
 * Announces from a relay feed the same delivery as phase 1.
 ******************************************/

// TaskSubscribeTagRelay is the queue task that makes @application follow a tag's relay
const TaskSubscribeTagRelay = "SubscribeTagRelay"

// TaskUnsubscribeTagRelay is the queue task that stops following a tag's relay once nobody here follows the tag
const TaskUnsubscribeTagRelay = "UnsubscribeTagRelay"

// TaskBackfillTagFollow is the queue task that copies a tag's recent relay posts into one User's timeline
const TaskBackfillTagFollow = "BackfillTagFollow"

// tagRelayBackfillMax is the most recent posts copied into a timeline when a User follows a tag
const tagRelayBackfillMax = 20

// TagRelay manages the @application actor's subscriptions to #hashtag relays
type TagRelay struct {
	activityService  *ActivityStream
	domainService    *Domain
	followingService *Following
	outboxService    *Outbox2
	tagFollowService *TagFollow
}

// NewTagRelay returns a fully initialized TagRelay service
func NewTagRelay() TagRelay {
	return TagRelay{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service
func (service *TagRelay) Refresh(factory *Factory) {
	service.activityService = factory.ActivityStream()
	service.domainService = factory.Domain()
	service.followingService = factory.Following()
	service.outboxService = factory.Outbox2()
	service.tagFollowService = factory.TagFollow()
}

/******************************************
 * Common Data Methods
 ******************************************/

// collection returns the TagRelay collection for the provided database session
func (service *TagRelay) collection(session data.Session) data.Collection {
	return session.Collection("TagRelay")
}

// Load retrieves the one TagRelay that matches the criteria
func (service *TagRelay) Load(session data.Session, criteria exp.Expression, tagRelay *model.TagRelay) error {

	if err := service.collection(session).Load(notDeleted(criteria), tagRelay); err != nil {
		return derp.Wrap(err, "service.TagRelay.Load", "Loading TagRelay", criteria)
	}

	return nil
}

// Save adds or updates a TagRelay in the database
func (service *TagRelay) Save(session data.Session, tagRelay *model.TagRelay, note string) error {

	if err := service.collection(session).Save(tagRelay, note); err != nil {
		return derp.Wrap(err, "service.TagRelay.Save", "Saving TagRelay", tagRelay.Value)
	}

	return nil
}

// LoadByValue retrieves the TagRelay for a normalized tag value
func (service *TagRelay) LoadByValue(session data.Session, value string, tagRelay *model.TagRelay) error {
	return service.Load(session, exp.Equal("value", value), tagRelay)
}

// LoadByActorURL retrieves the TagRelay whose relay actor has this URL
func (service *TagRelay) LoadByActorURL(session data.Session, actorURL string, tagRelay *model.TagRelay) error {
	return service.Load(session, exp.Equal("actorUrl", actorURL), tagRelay)
}

// LoadByFollowID retrieves the TagRelay whose Follow activity has this ID
func (service *TagRelay) LoadByFollowID(session data.Session, followID string, tagRelay *model.TagRelay) error {
	return service.Load(session, exp.Equal("followId", followID), tagRelay)
}

/******************************************
 * Custom Methods
 ******************************************/

// Subscribe makes the @application actor follow the relay for a tag, unless it already does
// or nobody here follows the tag any more
func (service *TagRelay) Subscribe(session data.Session, value string, name string) error {

	const location = "service.TagRelay.Subscribe"

	// RULE: Only subscribe while someone still follows the tag
	if followed, err := service.tagFollowService.IsFollowedByAnyone(session, value); err != nil {
		return derp.Wrap(err, location, "Counting TagFollows", value)
	} else if !followed {
		return nil
	}

	// RULE: One subscription per tag
	existing := model.NewTagRelay()
	if err := service.LoadByValue(session, value, &existing); err == nil {
		return nil
	} else if !derp.IsNotFound(err) {
		return derp.Wrap(err, location, "Loading TagRelay", value)
	}

	// Find the relay's actor for this tag (the client stack turns "#name" into name@tags.pub)
	relayName := TagRelayName(name)
	actor, err := service.activityService.AppClient().Load("#" + relayName)

	if err != nil {
		return derp.Wrap(err, location, "Loading relay actor", relayName)
	}

	// RULE: The relay must answer with an Actor
	if !actor.IsActor() {
		return derp.NotFound(location, "Relay did not return an Actor", relayName)
	}

	// Record the subscription before sending, so the relay's Accept can find it
	tagRelay := model.NewTagRelay()
	tagRelay.Name = relayName
	tagRelay.Value = value
	tagRelay.ActorURL = actor.ID()
	tagRelay.FollowID = service.domainService.ActorID() + "#follows/" + tagRelay.TagRelayID.Hex()

	if err := service.Save(session, &tagRelay, "Subscribed"); err != nil {
		return derp.Wrap(err, location, "Saving TagRelay", value)
	}

	// Send the Follow through the delivery pipeline, after this transaction commits
	if err := service.outboxService.SendFollow(session, service.domainService.ActorID(), tagRelay.FollowID, tagRelay.ActorURL); err != nil {
		return derp.Wrap(err, location, "Sending Follow", tagRelay.ActorURL)
	}

	// Subscribe and save
	return nil
}

// Unsubscribe stops following the relay for a tag once nobody here follows the tag
func (service *TagRelay) Unsubscribe(session data.Session, value string) error {

	const location = "service.TagRelay.Unsubscribe"

	// RULE: Keep the subscription while anyone still follows the tag
	if followed, err := service.tagFollowService.IsFollowedByAnyone(session, value); err != nil {
		return derp.Wrap(err, location, "Counting TagFollows", value)
	} else if followed {
		return nil
	}

	// Find the subscription; none means there's nothing to undo
	tagRelay := model.NewTagRelay()
	if err := service.LoadByValue(session, value, &tagRelay); err != nil {
		if derp.IsNotFound(err) {
			return nil
		}
		return derp.Wrap(err, location, "Loading TagRelay", value)
	}

	// Tell the relay to stop, quoting the original Follow so it knows which one to reverse
	follow := mapof.Any{
		vocab.PropertyID:     tagRelay.FollowID,
		vocab.PropertyType:   vocab.ActivityTypeFollow,
		vocab.PropertyActor:  service.domainService.ActorID(),
		vocab.PropertyObject: tagRelay.ActorURL,
	}

	if err := service.outboxService.SendUndoFollow(session, service.domainService.ActorID(), follow); err != nil {
		return derp.Wrap(err, location, "Sending Undo", tagRelay.ActorURL)
	}

	// Forget the subscription
	if err := service.collection(session).HardDelete(exp.Equal("_id", tagRelay.TagRelayID)); err != nil {
		return derp.Wrap(err, location, "Deleting TagRelay", value)
	}

	// Unsubscribed, unbothered
	return nil
}

// Accept marks a subscription active when its relay accepts the Follow
func (service *TagRelay) Accept(session data.Session, followID string, actorURL string) error {

	const location = "service.TagRelay.Accept"

	tagRelay := model.NewTagRelay()
	if err := service.LoadByFollowID(session, followID, &tagRelay); err != nil {
		return derp.Wrap(err, location, "Loading TagRelay", followID)
	}

	// RULE: Only the relay that was followed may accept
	if tagRelay.ActorURL != actorURL {
		return derp.Forbidden(location, "Accept is not from the followed relay", tagRelay.ActorURL, actorURL)
	}

	tagRelay.State = model.TagRelayStateActive

	if err := service.Save(session, &tagRelay, "Accepted"); err != nil {
		return derp.Wrap(err, location, "Saving TagRelay", followID)
	}

	// Accepted. Like a cool kid.
	return nil
}

// Reject removes a subscription when its relay rejects the Follow
func (service *TagRelay) Reject(session data.Session, followID string, actorURL string) error {

	const location = "service.TagRelay.Reject"

	tagRelay := model.NewTagRelay()
	if err := service.LoadByFollowID(session, followID, &tagRelay); err != nil {
		return derp.Wrap(err, location, "Loading TagRelay", followID)
	}

	// RULE: Only the relay that was followed may reject
	if tagRelay.ActorURL != actorURL {
		return derp.Forbidden(location, "Reject is not from the followed relay", tagRelay.ActorURL, actorURL)
	}

	if err := service.collection(session).HardDelete(exp.Equal("_id", tagRelay.TagRelayID)); err != nil {
		return derp.Wrap(err, location, "Deleting TagRelay", followID)
	}

	// Rejection builds character
	return nil
}

// Backfill copies a tag's most recent relay posts into one User's home timeline
func (service *TagRelay) Backfill(session data.Session, userID primitive.ObjectID, name string) error {

	const location = "service.TagRelay.Backfill"

	// Load the relay's actor and its outbox
	relayName := TagRelayName(name)
	actor, err := service.activityService.AppClient().Load("#" + relayName)

	if err != nil {
		return derp.Wrap(err, location, "Loading relay actor", relayName)
	}

	outbox, err := actor.Outbox().Load()

	if err != nil {
		return derp.Wrap(err, location, "Loading relay outbox", relayName)
	}

	// Copy each recent public post that really carries the tag
	value := model.ToToken(name)

	for item := range collections.RangeDocuments(outbox, collections.WithMaxDocuments(tagRelayBackfillMax)) {

		post, err := relayPost(item)

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Loading relayed post", item.ID()))
			continue
		}

		if !IsPublicPost(post) || !slices.Contains(DocumentTagValues(post), value) {
			continue
		}

		if err := service.followingService.SaveTaggedNewsItem(session, userID, name, post); err != nil {
			derp.Report(derp.Wrap(err, location, "Saving news item", userID, post.ID()))
		}
	}

	// Back to the future
	return nil
}

/******************************************
 * Helpers
 ******************************************/

// TagRelayName returns the name a relay knows a tag by: lowercase, without the leading #
func TagRelayName(name string) string {
	return strings.ToLower(TagFollowName(name))
}

// relayPost returns the post a relay item points to: the object of an Announce, or the item itself
func relayPost(item streams.Document) (streams.Document, error) {

	if item.Type() == vocab.ActivityTypeAnnounce {
		return item.Object().Load()
	}

	return item.Load()
}
