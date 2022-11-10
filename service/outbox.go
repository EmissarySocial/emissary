package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
)

// Outbox manages all interactions with a user's Outbox
type Outbox struct {
	collection data.Collection
}

// NewOutbox returns a fully populated Outbox service
func NewOutbox(collection data.Collection) Outbox {
	service := Outbox{
		collection: collection,
	}

	service.Refresh(collection)
	return service
}

/*******************************************
 * LIFECYCLE METHODS
 *******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Outbox) Refresh(collection data.Collection) {
	service.collection = collection
}

// Close stops any background processes controlled by this service
func (service *Outbox) Close() {

}

/*******************************************
 * COMMON DATA METHODS
 *******************************************/

// New creates a newly initialized Outbox that is ready to use
func (service *Outbox) New() model.OutboxItem {
	return model.NewOutboxItem()
}

// List returns an iterator containing all of the Outboxs who match the provided criteria
func (service *Outbox) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(notDeleted(criteria), options...)
}

// Load retrieves an Outbox from the database
func (service *Outbox) Load(criteria exp.Expression, result *model.OutboxItem) error {

	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Outbox", "Error loading Outbox", criteria)
	}

	return nil
}

// Save adds/updates an Outbox in the database
func (service *Outbox) Save(inboxItem *model.OutboxItem, note string) error {

	if err := service.collection.Save(inboxItem, note); err != nil {
		return derp.Wrap(err, "service.Outbox", "Error saving Outbox", inboxItem, note)
	}

	return nil
}

// Delete removes an Outbox from the database (virtual delete)
func (service *Outbox) Delete(inboxItem *model.OutboxItem, note string) error {

	// Delete Outbox record last.
	if err := service.collection.Delete(inboxItem, note); err != nil {
		return derp.Wrap(err, "service.Outbox", "Error deleting Outbox", inboxItem, note)
	}

	return nil
}
