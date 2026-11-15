package service

import (
	"slices"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/assanitizer"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
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

// DeleteActivity removes the activities that published an object from an actor's outbox, and
// publishes a Delete that is addressed as the object was
func (service *Outbox2) DeleteActivity(session data.Session, actorType string, actorID primitive.ObjectID, objectID string, permissions model.Permissions) error {

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
	addressing, err := service.deleteAddressing(session, actorType, actorID, objectID, permissions)

	if err != nil {
		return derp.Wrap(err, location, "Reading original addressing", objectID)
	}

	// Remove the activities that published this object from both outboxes
	if err := service.removeByObjectID(session, actorType, actorID, objectID); err != nil {
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
	embedded, err := assanitizer.WithoutBlindRecipients(originalActivity)

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

// deleteAddressing returns the addressing for a Delete of the provided object: the addressing of
// the newest activity that published it, or one built from its permissions when no such row exists
func (service *Outbox2) deleteAddressing(session data.Session, actorType string, actorID primitive.ObjectID, objectID string, permissions model.Permissions) (mapof.Any, error) {

	const location = "service.Outbox2.deleteAddressing"

	followersURL := service.locator.FollowersURL(actorType, actorID)

	// Find the newest Create or Update of this object
	originals, err := service.Query(session, publishedObjectCriteria(actorType, actorID, objectID), option.SortDesc("createDate"), option.MaxRows(1))

	if err != nil {
		return nil, derp.Wrap(err, location, "Loading original activity", objectID)
	}

	// An object published through the legacy outbox has no row here to copy from
	if len(originals) == 0 {
		return addressingFromPermissions(permissions, followersURL), nil
	}

	result := mapof.NewAny()
	copyAddressing(result, originals[0].Activity)
	addFollowersToPublic(result, followersURL)

	return result, nil
}

// addressingFromPermissions returns the addressing for an activity about an object whose own
// addressing was never stored: Public and the actor's followers when anyone may read the
// object, and no one otherwise
func addressingFromPermissions(permissions model.Permissions, followersURL string) mapof.Any {

	if permissions.IsAnonymous() {
		return mapof.Any{
			vocab.PropertyTo: []string{vocab.NamespacePublic},
			vocab.PropertyCC: []string{followersURL},
		}
	}

	// RULE: Delivery follows addressing alone, so naming the followers of a restricted object
	// would tell every follower about it. Its writer must supply the audience.
	return mapof.NewAny()
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
