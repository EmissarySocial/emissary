package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SearchTag defines a service that manages all searchable tags in a domain.
type SearchTag struct {
	collection data.Collection
	host       string
}

// NewSearchTag returns a fully initialized SearchTag service
func NewSearchTag() SearchTag {
	return SearchTag{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *SearchTag) Refresh(collection data.Collection, host string) {
	service.collection = collection
	service.host = host
}

// Close stops any background processes controlled by this service
func (service *SearchTag) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *SearchTag) Count(criteria exp.Expression) (int64, error) {
	return service.collection.Count(notDeleted(criteria))
}

// Query returns an slice of allthe SearchTags that match the provided criteria
func (service *SearchTag) Query(criteria exp.Expression, options ...option.Option) ([]model.SearchTag, error) {
	result := make([]model.SearchTag, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)

	return result, err
}

// List returns an iterator containing all of the SearchTags that match the provided criteria
func (service *SearchTag) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Load retrieves an SearchTag from the database
func (service *SearchTag) Load(criteria exp.Expression, searchTag *model.SearchTag) error {

	if err := service.collection.Load(notDeleted(criteria), searchTag); err != nil {
		return derp.Wrap(err, "service.SearchTag.Load", "Error loading SearchTag", criteria)
	}

	return nil
}

// Save adds/updates an SearchTag in the database
func (service *SearchTag) Save(searchTag *model.SearchTag, note string) error {

	// Validate the value before saving
	if err := service.Schema().Validate(searchTag); err != nil {
		return derp.Wrap(err, "service.SearchTag.Save", "Error validating SearchTag", searchTag)
	}

	// Save the searchTag to the database
	if err := service.collection.Save(searchTag, note); err != nil {
		return derp.Wrap(err, "service.SearchTag.Save", "Error saving SearchTag", searchTag, note)
	}

	return nil
}

// Delete removes an SearchTag from the database (virtual delete)
func (service *SearchTag) Delete(searchTag *model.SearchTag, note string) error {

	// Delete this SearchTag
	if err := service.collection.Delete(searchTag, note); err != nil {
		return derp.Wrap(err, "service.SearchTag.Delete", "Error deleting SearchTag", searchTag, note)
	}

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *SearchTag) ObjectType() string {
	return "SearchTag"
}

// New returns a fully initialized model.Group as a data.Object.
func (service *SearchTag) ObjectNew() data.Object {
	result := model.NewSearchTag()
	return &result
}

func (service *SearchTag) ObjectID(object data.Object) primitive.ObjectID {

	if mention, ok := object.(*model.SearchTag); ok {
		return mention.SearchTagID
	}

	return primitive.NilObjectID
}

func (service *SearchTag) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *SearchTag) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

func (service *SearchTag) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewSearchTag()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *SearchTag) ObjectSave(object data.Object, comment string) error {
	if searchTag, ok := object.(*model.SearchTag); ok {
		return service.Save(searchTag, comment)
	}
	return derp.NewInternalError("service.SearchTag.ObjectSave", "Invalid Object Type", object)
}

func (service *SearchTag) ObjectDelete(object data.Object, comment string) error {
	if searchTag, ok := object.(*model.SearchTag); ok {
		return service.Delete(searchTag, comment)
	}
	return derp.NewInternalError("service.SearchTag.ObjectDelete", "Invalid Object Type", object)
}

func (service *SearchTag) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.NewUnauthorizedError("service.SearchTag", "Not Authorized")
}

func (service *SearchTag) Schema() schema.Schema {
	return schema.New(model.SearchTagSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *SearchTag) LoadByID(searchTagID primitive.ObjectID, searchTag *model.SearchTag) error {
	criteria := exp.Equal("_id", searchTagID)
	return service.Load(criteria, searchTag)
}

func (service *SearchTag) LoadByName(name string, searchTag *model.SearchTag) error {
	criteria := exp.Equal("name", name)
	return service.Load(criteria, searchTag)
}

// Upsert verifies that a SearchTag exists in the database, and creates it if it does not.
func (service *SearchTag) Upsert(name string) error {

	// Try to find the SearchTag in the database
	searchTag := model.NewSearchTag()
	err := service.LoadByName(name, &searchTag)

	// If it exists, then we're done
	if err == nil {
		return nil
	}

	// If "not found" then create a new SearchTag``
	if derp.NotFound(err) {

		// Set default values for the new SearchTag
		searchTag.Name = name
		searchTag.StateID = model.SearchTagStateWaiting

		if err := service.Save(&searchTag, "New SearchTag"); err != nil {
			return derp.Wrap(err, "service.SearchTag.Upsert", "Error saving SearchTag", name)
		}

		return nil
	}

	// Otherwise, return the error to the caller. (This should never happen)
	return derp.Wrap(err, "service.SearchTag.Upsert", "Error loading SearchTag", name)
}
