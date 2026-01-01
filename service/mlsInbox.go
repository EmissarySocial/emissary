package service

import (
	"iter"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"
)

// MLSInbox manages all interactions with the MLSInbox collection
type MLSInbox struct{}

// NewMLSInbox returns a fully populated MLSInbox service
func NewMLSInbox() MLSInbox {
	return MLSInbox{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *MLSInbox) Refresh() {

}

// Close stops any background processes controlled by this service
func (service *MLSInbox) Close() {

}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *MLSInbox) collection(session data.Session) data.Collection {
	return session.Collection("MLSInbox")
}

// Count returns the number of records that match the provided criteria
func (service *MLSInbox) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns a slice of MLSInboxs that match the provided criteria
func (service *MLSInbox) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.MLSMessage, error) {
	result := make([]model.MLSMessage, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator containing all of the MLSInboxs who match the provided criteria
func (service *MLSInbox) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the MLSInboxs that match the provided criteria
func (service *MLSInbox) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.MLSMessage], error) {

	iter, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.MLSInbox.Range", "Unable to create iterator", criteria)
	}

	return RangeFunc(iter, model.NewMLSMessage), nil
}

// Load retrieves an MLSInbox from the database
func (service *MLSInbox) Load(session data.Session, criteria exp.Expression, result *model.MLSMessage) error {

	const location = "service.MLSInbox.Load"

	if err := service.collection(session).Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, location, "Unable to load MLSInbox", criteria)
	}

	return nil
}

// Save adds/updates an MLSInbox in the database
func (service *MLSInbox) Save(session data.Session, circle *model.MLSMessage, note string) error {

	const location = "service.MLSInbox.Save"

	// Validate the value before saving
	if err := service.Schema().Validate(circle); err != nil {
		return derp.Wrap(err, location, "Unable to validate MLSInbox", circle)
	}

	// Save the value to the database
	if err := service.collection(session).Save(circle, note); err != nil {
		return derp.Wrap(err, location, "Unable to save MLSInbox", circle, note)
	}

	return nil
}

// Delete removes an MLSInbox from the database (virtual delete)
func (service *MLSInbox) Delete(session data.Session, circle *model.MLSMessage, note string) error {

	const location = "service.MLSInbox.Delete"

	if err := service.collection(session).Delete(circle, note); err != nil {
		return derp.Wrap(err, location, "Unable to delete MLSInbox", circle, note)
	}

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

func (service *MLSInbox) Schema() schema.Schema {
	return schema.New(model.MLSMessageSchema())
}

/******************************************
 * Custom Queries
 ******************************************/
