package service

import (
	"html/template"
	"sync"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/maps"
)

// Domain service manages all access to the singleton model.Domain in the database
type Domain struct {
	collection data.Collection
	funcMap    template.FuncMap
	model      model.Domain
	lock       *sync.Mutex
}

// NewDomain returns a fully initialized Domain service
func NewDomain(collection data.Collection, funcMap template.FuncMap) Domain {
	service := Domain{
		funcMap: funcMap,
		lock:    &sync.Mutex{},
	}

	service.Refresh(collection)

	return service
}

/*******************************************
 * LIFECYCLE METHODS
 *******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Domain) Refresh(collection data.Collection) {
	service.collection = collection
	service.model = model.NewDomain()
}

// Close stops the subscription service watcher
func (service *Domain) Close() {
}

/*******************************************
 * COMMON DATA METHODS
 *******************************************/

// Load retrieves an Domain from the database (or in-memory cache)
func (service *Domain) Load(domain *model.Domain) error {

	// If the value is already cached, then return it
	if !service.model.DomainID.IsZero() {
		*domain = service.model
		return nil
	}

	// Initialize a new object (to avoid NPE errors)
	service.lock.Lock()
	defer service.lock.Unlock()

	service.model = model.NewDomain()

	// If not cached, try to load from database
	err := service.collection.Load(exp.All(), &service.model)

	// If present in database, return success
	if err == nil {
		*domain = service.model
		return nil
	}

	// If not in database, try to create a new record
	if derp.NotFound(err) {

		if err := service.Save(domain, "Create New Domain"); err != nil {
			return derp.Wrap(err, "service.Domain.Load", "Error creating new domain")
		}

		*domain = service.model
		return nil
	}

	// Otherwise, there's some bigger error happening, fail un-gracefully
	return derp.Wrap(err, "service.Domain.Load", "Error loading Domain")
}

// Save updates the value of this domain in the database (and in-memory cache)
func (service *Domain) Save(domain *model.Domain, note string) error {

	// Try to save the value to the database
	if err := service.collection.Save(domain, note); err != nil {
		return derp.Wrap(err, "service.Domain.Save", "Error saving Domain")
	}

	// Update the in-memory cache
	service.model = *domain

	return nil
}

/*******************************************
 * GENERIC DATA METHODS
 *******************************************/

// New returns a fully initialized model.Stream as a data.Object.
func (service *Domain) ObjectNew() data.Object {
	result := model.NewDomain()
	return &result
}

func (service *Domain) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return nil, derp.NewBadRequestError("service.Domain.ObjectDelete", "Unsupported")
}

func (service *Domain) ObjectLoad(_ exp.Expression) (data.Object, error) {
	result := model.NewDomain()
	err := service.Load(&result)
	return &result, err
}

func (service *Domain) ObjectSave(object data.Object, note string) error {
	return service.Save(object.(*model.Domain), note)
}

func (service *Domain) ObjectDelete(object data.Object, note string) error {
	return derp.NewBadRequestError("service.Domain.ObjectDelete", "Unsupported")
}

func (service *Domain) Debug() maps.Map {
	return maps.Map{
		"service": "Domain",
	}
}
