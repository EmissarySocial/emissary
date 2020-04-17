package service

import (
	"github.com/benpate/data"
	"github.com/benpate/data/expression"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CollectionPage is the database collection where Pages are stored
const CollectionPage = "Page"

// Page manages all interactions with the Page collection
type Page struct {
	factory *Factory
	session data.Session
}

// New creates a newly initialized Page that is ready to use
func (service Page) New() *model.Page {
	return &model.Page{
		PageID: primitive.NewObjectID(),
	}
}

// List returns an iterator containing all of the Pages who match the provided criteria
func (service Page) List(criteria expression.Expression, options ...option.Option) (data.Iterator, *derp.Error) {

	return nil, nil
}

// Load retrieves an Page from the database
func (service Page) Load(criteria expression.Expression) (*model.Page, *derp.Error) {

	page := service.New()

	if err := service.session.Load(CollectionPage, criteria, page); err != nil {
		return nil, derp.Wrap(err, "service.Page", "Error loading Page", criteria)
	}

	return page, nil
}

// Save adds/updates an Page in the database
func (service Page) Save(page *model.Page, note string) *derp.Error {

	if err := service.session.Save(CollectionPage, page, note); err != nil {
		return derp.Wrap(err, "service.Page", "Error saving Page", page, note)
	}

	return nil
}

// Delete removes an Page from the database (virtual delete)
func (service Page) Delete(page *model.Page, note string) *derp.Error {

	if err := service.session.Delete(CollectionPage, page, note); err != nil {
		return derp.Wrap(err, "service.Page", "Error deleting Page", page, note)
	}

	return nil
}

//// GENERIC FUNCTIONS //////////////////

// NewObject wraps the `New` method as a generic Object
func (service Page) NewObject() data.Object {
	return service.New()
}

// ListObjects wraps the `List` method as a generic Object
func (service Page) ListObjects(criteria expression.Expression, options ...option.Option) (data.Iterator, *derp.Error) {
	return service.List(criteria, options...)
}

// LoadObject wraps the `Load` method as a generic Object
func (service Page) LoadObject(criteria expression.Expression) (data.Object, *derp.Error) {
	return service.Load(criteria)
}

// SaveObject wraps the `Save` method as a generic Object
func (service Page) SaveObject(object data.Object, note string) *derp.Error {

	if object, ok := object.(*model.Page); ok {
		return service.Save(object, note)
	}

	// This should never happen.
	return derp.New(derp.CodeInternalError, "service.Page", "Object is not a model.Page", object, note)
}

// DeleteObject wraps the `Delete` method as a generic Object
func (service Page) DeleteObject(object data.Object, note string) *derp.Error {

	if object, ok := object.(*model.Page); ok {
		return service.Delete(object, note)
	}

	// This should never happen.
	return derp.New(derp.CodeInternalError, "service.Page", "Object is not a model.Page", object, note)
}

// Close cleans up the service and any outstanding connections.
func (service Page) Close() {
	service.session.Close()
}
