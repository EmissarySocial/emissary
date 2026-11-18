package service

import (
	"slices"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal"
	"github.com/benpate/hannibal/datetime"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// legacyOutboxRemover removes rows from the legacy Outbox collection, which still holds
// activities published before their writer moved to Outbox2
type legacyOutboxRemover interface {
	removeOutboxMessagesByObjectID(session data.Session, actorType string, actorID primitive.ObjectID, objectID string) error
	removeOutboxMessagesByActivityURL(session data.Session, actorType string, actorID primitive.ObjectID, activityURL string) error
}

// documentDeleter removes one document from the ActivityStream cache
type documentDeleter interface {
	Delete(url string) error
}

// addressingProperties lists the properties that name an activity's recipients
var addressingProperties = []string{vocab.PropertyTo, vocab.PropertyCC, vocab.PropertyBTo, vocab.PropertyBCC}

// DeleteActivity removes the activities that published an object from an actor's outbox, and publishes
// a Delete addressed as the object was, or to audience when no Create or Update of it is stored here
func (service *Outbox2) DeleteActivity(session data.Session, actorType string, actorID primitive.ObjectID, objectID string, permissions model.Permissions, audience mapof.Any) error {

	const location = "service.Outbox2.DeleteActivity"

	// RULE: Only an actor with an outbox can delete from it
	actorURL := service.locator.ActorURL(actorType, actorID)

	if actorURL == "" {
		return derp.Internal(location, "Actor type has no outbox", "actorType: "+actorType)
	}

	// RULE: A Delete must name the object it deletes
	if objectID == "" {
		return derp.Internal(location, "ObjectID must not be empty", "actorId: "+actorID.Hex())
	}

	// Read the original addressing BEFORE removing the rows that hold it
	addressing, err := service.deleteAddressing(session, actorType, actorID, objectID, audience)

	if err != nil {
		return derp.Wrap(err, location, "Reading original addressing", objectID)
	}

	// Remove the activities that published this object from both outboxes
	if err := service.RemoveByObjectID(session, actorType, actorID, objectID); err != nil {
		return derp.Wrap(err, location, "Removing outbox activities", objectID)
	}

	if err := service.legacyOutbox.removeOutboxMessagesByObjectID(session, actorType, actorID, objectID); err != nil {
		return derp.Wrap(err, location, "Removing legacy outbox messages", objectID)
	}

	// Remove the object from the cache, so that no reader is served the deleted copy
	if err := service.objectCache.Delete(objectID); err != nil {
		return derp.Wrap(err, location, "Removing object from cache", objectID)
	}

	// Build the Delete. It has no record of its own, so Publish mints its id.
	activity := mapof.Any{
		vocab.AtContext:     vocab.ContextTypeActivityStreams,
		vocab.PropertyType:  vocab.ActivityTypeDelete,
		vocab.PropertyActor: actorURL,
		vocab.PropertyObject: mapof.Any{
			vocab.PropertyID:   objectID,
			vocab.PropertyType: vocab.ObjectTypeTombstone,
		},
		vocab.PropertyPublished: datetime.Now(),
	}

	copyAddressing(activity, addressing)

	if _, err := service.Publish(session, actorType, actorID, streams.NewDocument(activity), permissions); err != nil {
		return derp.Wrap(err, location, "Publishing Delete activity", objectID)
	}

	// Gone, but not forgotten
	return nil
}

// UndoActivity removes an activity from an actor's outbox, and publishes an Undo that embeds it
// and is addressed as it was
func (service *Outbox2) UndoActivity(session data.Session, actorType string, actorID primitive.ObjectID, originalActivity mapof.Any, permissions model.Permissions) error {

	const location = "service.Outbox2.UndoActivity"

	// RULE: Only an actor with an outbox can undo from it
	actorURL := service.locator.ActorURL(actorType, actorID)

	if actorURL == "" {
		return derp.Internal(location, "Actor type has no outbox", "actorType: "+actorType)
	}

	// RULE: The activity to undo is identified by its own URL
	activityURL := originalActivity.GetString(vocab.PropertyID)

	if activityURL == "" {
		return derp.Internal(location, "Original activity must have an id", "actorId: "+actorID.Hex())
	}

	// Remove the original activity from both outboxes
	if err := service.removeByActivityURL(session, actorType, actorID, activityURL); err != nil {
		return derp.Wrap(err, location, "Removing outbox activities", activityURL)
	}

	if err := service.legacyOutbox.removeOutboxMessagesByActivityURL(session, actorType, actorID, activityURL); err != nil {
		return derp.Wrap(err, location, "Removing legacy outbox messages", activityURL)
	}

	// Embed the caller's copy of the original, because its row may already be gone (RULES P7-2).
	// It is copied first, so that stripping the blind recipients leaves the caller's map intact.
	embedded, err := hannibal.StripBlindRecipients(originalActivity)

	if err != nil {
		return derp.Wrap(err, location, "Removing blind recipients", activityURL)
	}

	// Build the Undo. It has no record of its own, so Publish mints its id.
	activity := mapof.Any{
		vocab.AtContext:         vocab.ContextTypeActivityStreams,
		vocab.PropertyType:      vocab.ActivityTypeUndo,
		vocab.PropertyActor:     actorURL,
		vocab.PropertyObject:    embedded,
		vocab.PropertyPublished: datetime.Now(),
	}

	// RULE: An Undo reaches whoever received the original, so an author-only Like makes an author-only Undo
	copyAddressing(activity, originalActivity)

	if _, err := service.Publish(session, actorType, actorID, streams.NewDocument(activity), permissions); err != nil {
		return derp.Wrap(err, location, "Publishing Undo activity", activityURL)
	}

	// Ctrl+Z
	return nil
}

// PublishAnnounce stores and delivers a public "Announce" of object by a local actor, addressed to
// Public and that actor's followers, and returns the Announce's URL
func (service *Outbox2) PublishAnnounce(session data.Session, actorType string, actorID primitive.ObjectID, actorURL string, object streams.Document) (string, error) {

	const location = "service.Outbox2.PublishAnnounce"

	// A boost reaches the booster's followers and Public, never the original's private audience.
	// It carries no id, so Publish mints one that can be fetched from the actor's outbox.
	announce := streams.NewDocument(mapof.Any{
		vocab.AtContext:         vocab.ContextTypeActivityStreams,
		vocab.PropertyType:      vocab.ActivityTypeAnnounce,
		vocab.PropertyActor:     actorURL,
		vocab.PropertyObject:    object.Map(),
		vocab.PropertyPublished: datetime.Now(),
		vocab.PropertyTo:        []string{vocab.NamespacePublic},
		vocab.PropertyCC:        []string{service.locator.FollowersURL(actorType, actorID)},
	})

	activityURL, err := service.Publish(session, actorType, actorID, announce, model.NewAnonymousPermissions())

	if err != nil {
		return "", derp.Wrap(err, location, "Publishing Announce", "actorId: "+actorID.Hex())
	}

	// Boosted. To infinity, and beyond!
	return activityURL, nil
}

// deleteAddressing returns the addressing for a Delete of the provided object: the addressing of
// the newest activity that published it, or the caller's audience when no such row exists
func (service *Outbox2) deleteAddressing(session data.Session, actorType string, actorID primitive.ObjectID, objectID string, audience mapof.Any) (mapof.Any, error) {

	const location = "service.Outbox2.deleteAddressing"

	followersURL := service.locator.FollowersURL(actorType, actorID)

	// Find the newest Create or Update of this object
	originals, err := service.Query(session, publishedObjectCriteria(actorType, actorID, objectID), option.SortDesc("createDate"), option.MaxRows(1))

	if err != nil {
		return nil, derp.Wrap(err, location, "Loading original activity", objectID)
	}

	// RULE: An object published through the legacy outbox has no row here to copy from. Its writer
	// knows who it was shared with, and permissions alone cannot name a circle.
	result := mapof.NewAny()

	if len(originals) == 0 {
		copyAddressing(result, audience)
		return result, nil
	}

	copyAddressing(result, originals[0].Activity)
	addFollowersToPublic(result, followersURL)

	return result, nil
}

// addFollowersToPublic adds the actor's followers collection to `cc` when the addressing names
// Public and does not name the collection already
func addFollowersToPublic(addressing mapof.Any, followersURL string) {

	recipients := make([]string, 0)

	for _, name := range addressingProperties {
		recipients = append(recipients, addressing.GetSliceOfString(name)...)
	}

	// RULE: Followers learn of a Delete only when the original was theirs to see. Adding them to
	// a private activity would tell every follower the id of a post they never received.
	if !slices.ContainsFunc(recipients, isPublicURI) {
		return
	}

	if slices.Contains(recipients, followersURL) {
		return
	}

	addressing[vocab.PropertyCC] = append(addressing.GetSliceOfString(vocab.PropertyCC), followersURL)
}

// copyAddressing copies each addressing property that the source holds onto the target
func copyAddressing(target mapof.Any, source mapof.Any) {

	for _, name := range addressingProperties {

		recipients := source.GetSliceOfString(name)

		if len(recipients) == 0 {
			continue
		}

		target[name] = slices.Clone(recipients)
	}
}
