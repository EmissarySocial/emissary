package service

import (
	"github.com/benpate/data"
	"github.com/benpate/data/expression"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CollectionDomain is the database collection where Domains are stored
const CollectionDomain = "Domain"

// Domain manages all interactions with the Domain collection
type Domain struct {
	factory *Factory
	session data.Session
}

// New creates a newly initialized Domain that is ready to use
func (service Domain) New() *model.Domain {
	return &model.Domain{
		DomainID: primitive.NewObjectID().Hex(),
	}
}

// List returns an iterator containing all of the Domains who match the provided criteria
func (service Domain) List(criteria expression.Expression, options ...option.Option) (data.Iterator, *derp.Error) {

	return nil, nil
}

// Load retrieves an Domain from the database
func (service Domain) Load(criteria expression.Expression) (*model.Domain, *derp.Error) {

	domain := service.New()

	if err := service.session.Load(CollectionDomain, criteria, domain); err != nil {
		return nil, derp.Wrap(err, "service.Domain", "Error loading Domain", criteria)
	}

	return domain, nil
}

// Save adds/updates an Domain in the database
func (service Domain) Save(domain *model.Domain, note string) *derp.Error {

	if err := service.session.Save(CollectionDomain, domain, note); err != nil {
		return derp.Wrap(err, "service.Domain", "Error saving Domain", domain, note)
	}

	return nil
}

// Delete removes an Domain from the database (virtual delete)
func (service Domain) Delete(domain *model.Domain, note string) *derp.Error {

	if err := service.session.Delete(CollectionDomain, domain, note); err != nil {
		return derp.Wrap(err, "service.Domain", "Error deleting Domain", domain, note)
	}

	return nil
}

//// GENERIC FUNCTIONS //////////////////

// NewObject wraps the `New` method as a generic Object
func (service Domain) NewObject() data.Object {
	return service.New()
}

// ListObjects wraps the `List` method as a generic Object
func (service Domain) ListObjects(criteria expression.Expression, options ...option.Option) (data.Iterator, *derp.Error) {
	return service.List(criteria, options...)
}

// LoadObject wraps the `Load` method as a generic Object
func (service Domain) LoadObject(criteria expression.Expression) (data.Object, *derp.Error) {
	return service.Load(criteria)
}

// SaveObject wraps the `Save` method as a generic Object
func (service Domain) SaveObject(object data.Object, note string) *derp.Error {

	if object, ok := object.(*model.Domain); ok {
		return service.Save(object, note)
	}

	// This should never happen.
	return derp.New(derp.CodeInternalError, "service.Domain", "Object is not a model.Domain", object, note)
}

// DeleteObject wraps the `Delete` method as a generic Object
func (service Domain) DeleteObject(object data.Object, note string) *derp.Error {

	if object, ok := object.(*model.Domain); ok {
		return service.Delete(object, note)
	}

	// This should never happen.
	return derp.New(derp.CodeInternalError, "service.Domain", "Object is not a model.Domain", object, note)
}

// Close cleans up the service and any outstanding connections.
func (service Domain) Close() {
	service.session.Close()
}
