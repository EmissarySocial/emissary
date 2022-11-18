package activitypub

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"sync"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/go-fed/activity/streams/vocab"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Database struct {
	factory Factory

	// Enables mutations. A sync.Mutex per ActivityPub ID.
	locks *sync.Map

	// The host domain of our service, for detecting ownership.
	hostname string
}

func NewDatabase(factory Factory, outboxService *service.Outbox, hostname string) *Database {
	return &Database{
		factory:  factory,
		locks:    &sync.Map{},
		hostname: hostname,
	}
}

func (db *Database) Lock(ctx context.Context, id *url.URL) error {
	// Before any other Database methods are called, the relevant `id`
	// entries are locked to allow for fine-grained concurrency.

	// Strategy: create a new lock, if stored, continue. Otherwise, lock the
	// existing mutex.
	mu := &sync.Mutex{}
	mu.Lock() // Optimistically lock if we do store it.
	i, loaded := db.locks.LoadOrStore(id.String(), mu)
	if loaded {
		mu = i.(*sync.Mutex)
		mu.Lock()
	}
	return nil
}

func (db *Database) Unlock(ctx context.Context, id *url.URL) error {
	// Once Go-Fed is done calling Database methods, the relevant `id`
	// entries are unlocked.

	i, ok := db.locks.Load(id.String())
	if !ok {
		return errors.New("Missing an id in Unlock")
	}
	mu := i.(*sync.Mutex)
	mu.Unlock()
	return nil
}

func (db *Database) Owns(ctx context.Context, itemURL *url.URL) (owns bool, err error) {
	// Owns just determines if the ActivityPub id is owned by this server.
	// TODO: In a real implementation, consider something far more robust than
	// this string comparison.
	return itemURL.Host == db.hostname, nil
}

func (db *Database) Exists(_ context.Context, itemURL *url.URL) (exists bool, err error) {

	const location = "activitypub.Database.Exists"

	_, _, internalError := db.load(itemURL)

	if internalError == nil {
		return true, nil
	}

	if derp.NotFound(internalError) {
		return false, nil
	}

	return false, derp.Wrap(internalError, "Database.Exists", "Error checking if object exists", itemURL.String())
}

func (db *Database) Get(_ context.Context, itemURL *url.URL) (value vocab.Type, err error) {

	const location = "activitypub.Database.Get"

	object, itemType, err := db.load(itemURL)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error loading object", itemURL)
	}

	// Convert the model object to an ActivityStream object
	return ToActivityStream(object, itemType)
}

func (db *Database) Create(_ context.Context, item vocab.Type) error {
	return db.save(item, "Create via ActivityPub")
}

func (db *Database) Update(_ context.Context, item vocab.Type) error {
	return db.save(item, "Update via ActivityPub")
}

func (db *Database) Delete(_ context.Context, itemURL *url.URL) error {

	const location = "service.activitypub.Database.Create"

	// Find the object in the database
	object, itemType, err := db.load(itemURL)

	if err != nil {
		return derp.Wrap(err, location, "Error loading object", itemURL)
	}

	// Get the corresponding ModelService to interact with the database
	modelService, err := db.factory.Model(itemType)

	if err != nil {
		return derp.Wrap(err, location, "No service found for item type", itemType)
	}

	// "Delete" the object from the database
	if err := modelService.ObjectDelete(object, "Delete via ActivityPub"); err != nil {
		return derp.Wrap(err, location, "Error deleting object", object)
	}

	// Success?!?
	return nil
}

func (db *Database) InboxContains(_ context.Context, inboxURL *url.URL, itemURL *url.URL) (contains bool, err error) {

	// Guarantee that the item's URL is contained within the inbox URL.
	if !strings.HasPrefix(itemURL.String(), inboxURL.String()) {
		return false, derp.NewBadRequestError("activitypub.Database.InboxContains", "Item URL does not match inbox URL", itemURL.String(), inboxURL.String())
	}

	return db.Exists(nil, itemURL)
}

func (db *Database) GetInbox(ctx context.Context, inboxURL *url.URL) (inbox vocab.ActivityStreamsOrderedCollectionPage, err error) {
	return db.getOrderedCollectionPage(ctx, inboxURL, ItemTypeInbox)
}

func (db *Database) SetInbox(ctx context.Context, inbox vocab.ActivityStreamsOrderedCollectionPage) error {

	for it := inbox.GetActivityStreamsOrderedItems().Begin(); it != nil; it = it.Next() {
		select {

		// check if context was cancelled
		case <-ctx.Done():
			return ctx.Err()

		// otherwise, save the next item
		default:

			item := it.GetType()
			_, itemType, _, err := parseItem(item)

			if err != nil {
				return derp.Wrap(err, "activitypub.Database.SetOutbox", "Error parsing item", item)
			}

			if itemType != ItemTypeInbox {
				return derp.NewBadRequestError("activitypub.Database.SetOutbox", "Item is not an outbox", item)
			}

			if err := db.save(item, "SetInbox via ActivityPub"); err != nil {
				return derp.Wrap(err, "activitypub.Database.SetInbox", "Error saving item", item)
			}
		}
	}

	return nil
}

func (db *Database) GetOutbox(ctx context.Context, outboxURL *url.URL) (outbox vocab.ActivityStreamsOrderedCollectionPage, err error) {
	return db.getOrderedCollectionPage(ctx, outboxURL, ItemTypeOutbox)
}

func (db *Database) SetOutbox(ctx context.Context, outbox vocab.ActivityStreamsOrderedCollectionPage) error {

	for it := outbox.GetActivityStreamsOrderedItems().Begin(); it != nil; it = it.Next() {
		select {

		// check if context was cancelled
		case <-ctx.Done():
			return ctx.Err()

		// otherwise, save the next item
		default:

			item := it.GetType()
			_, itemType, _, err := parseItem(item)

			if err != nil {
				return derp.Wrap(err, "activitypub.Database.SetOutbox", "Error parsing item", item)
			}

			if itemType != ItemTypeOutbox {
				return derp.NewBadRequestError("activitypub.Database.SetOutbox", "Item is not an outbox", item)
			}

			if err := db.save(item, "SetOutbox via ActivityPub"); err != nil {
				return derp.Wrap(err, "activitypub.Database.SetInbox", "Error saving item", item)
			}
		}
	}

	return nil
}

func (db *Database) ActorForOutbox(ctx context.Context, outboxURL *url.URL) (actorURL *url.URL, err error) {
	userID, _, _, err := parseURL(outboxURL)
	return service.ActorID(db.hostname, userID), nil
}

func (db *Database) ActorForInbox(ctx context.Context, outboxURL *url.URL) (actorURL *url.URL, err error) {
	userID, _, _, err := parseURL(outboxURL)
	return service.ActorID(db.hostname, userID), nil
}

func (db *Database) OutboxForInbox(ctx context.Context, inboxURL *url.URL) (outboxURL *url.URL, err error) {
	userID, _, _, err := parseURL(inboxURL)
	return service.ActorOutbox(db.hostname, userID), nil
}

func (db *Database) NewID(ctx context.Context, item vocab.Type) (id *url.URL, err error) {

	itemType := item.GetTypeName()
	itemID := primitive.NewObjectID()
	userID := getActorID(item)

	urlString := "/.activitypub/" + userID.Hex() + "/" + itemType + "/" + itemID.Hex()
	return url.Parse(urlString)

	// Generate a new `id` for the ActivityStreams object `t`.

	// You can be fancy and put different types authored by different folks
	// along different paths. Or just generate a GUID. Implementation here
	// is left as an exercise for the reader.
}

func (db *Database) Followers(ctx context.Context, actorURL *url.URL) (vocab.ActivityStreamsCollection, error) {
	return db.getCollection("follwers", actorURL, exp.All())
}

func (db *Database) Following(ctx context.Context, actorURL *url.URL) (vocab.ActivityStreamsCollection, error) {
	return db.getCollection("following", actorURL, exp.All())
}

func (db *Database) Liked(ctx context.Context, actorURL *url.URL) (vocab.ActivityStreamsCollection, error) {
	return db.getCollection("reaction", actorURL, exp.Equal("type", model.ReactionTypeLike))
}

/***********************************
 * Helper Methods
 ***********************************/

func (db *Database) load(id *url.URL) (data.Object, string, error) {

	const location = "activitypub.Database.load"

	userID, itemType, itemID, err := parseURL(id)

	if err != nil {
		return nil, itemType, derp.Wrap(err, location, "Error parsing URL", id)
	}

	// Get the service for this kind of item
	modelService, err := db.factory.Model(itemType)

	if err != nil {
		return nil, itemType, derp.Wrap(err, location, "No service found for item type", itemType)
	}

	// Try to load the record from the database
	object, err := modelService.ObjectLoad(exp.Equal("_id", itemID).AndEqual("userId", userID))

	if err != nil {
		return nil, itemType, derp.Wrap(err, location, "Error loading object", id)
	}

	return object, itemType, nil
}

func (db *Database) save(item vocab.Type, comment string) error {

	const location = "service.activitypub.Database.save"

	// Extract important values from the item ID
	_, itemType, _, err := parseItem(item)

	if err != nil {
		return derp.Wrap(err, location, "Error parsing Item", item)
	}

	// Get the service for this kind of item
	modelService, err := db.factory.Model(itemType)

	if err != nil {
		return derp.Wrap(err, location, "No service found for item type", itemType)
	}

	// Convert the ActivityStream object to a model object
	object, err := ToModelObject(item)

	if err != nil {
		return derp.Wrap(err, location, "Error converting item to model object", item)
	}

	// Save the object to the database
	if err := modelService.ObjectSave(object, comment); err != nil {
		return derp.Wrap(err, location, "Error saving object", object)
	}

	// Success?!?
	return nil
}
