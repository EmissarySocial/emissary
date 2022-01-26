package service

import (
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/whisperverse/whisperverse/model"
)

type Outbox struct {
	collection data.Collection
}

func NewOutbox(collection data.Collection) Outbox {
	return Outbox{
		collection: collection,
	}
}

/*******************************************
 * COMMON DATA FUNCTIONS
 *******************************************/

// List returns an iterator containing all of the Outboxes who match the provided criteria
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
func (service *Outbox) Save(user *model.OutboxItem, note string) error {

	if err := service.collection.Save(user, note); err != nil {
		return derp.Wrap(err, "service.Outbox", "Error saving Outbox", user, note)
	}

	return nil
}

// Delete removes an Outbox from the database (virtual delete)
func (service *Outbox) Delete(user *model.OutboxItem, note string) error {

	if err := service.collection.Delete(user, note); err != nil {
		return derp.Wrap(err, "service.Outbox", "Error deleting Outbox", user, note)
	}

	return nil
}

/*******************************************
 * GENERIC DATA FUNCTIONS
 *******************************************/

// New returns a fully initialized model.Stream as a data.Object.
func (service *Outbox) ObjectNew() data.Object {
	result := model.NewOutboxItem()
	return &result
}

func (service *Outbox) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

func (service *Outbox) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewOutboxItem()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Outbox) ObjectSave(object data.Object, comment string) error {
	return service.Save(object.(*model.OutboxItem), comment)
}

func (service *Outbox) ObjectDelete(object data.Object, comment string) error {
	return service.Delete(object.(*model.OutboxItem), comment)
}

func (service *Outbox) Debug() datatype.Map {
	return datatype.Map{
		"service": "Outbox",
	}
}
