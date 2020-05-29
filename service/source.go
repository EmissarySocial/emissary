package service

import (
	"github.com/benpate/data"
	"github.com/benpate/data/expression"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service/source"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CollectionSource is the database collection where Sources are stored
const CollectionSource = "Source"

// Source manages all interactions with the Source collection
type Source struct {
	factory Factory
	session data.Session
}

// New creates a newly initialized Source that is ready to use
func (service Source) New() *model.Source {

	return &model.Source{
		SourceID: primitive.NewObjectID(),
	}
}

// List returns an iterator containing all of the Sources who match the provided criteria
func (service Source) List(criteria expression.Expression, options ...option.Option) (data.Iterator, *derp.Error) {
	return service.session.List(CollectionSource, criteria, options...)
}

// Load retrieves an Source from the database
func (service Source) Load(criteria expression.Expression) (*model.Source, *derp.Error) {

	account := service.New()

	if err := service.session.Load(CollectionSource, criteria, account); err != nil {
		return nil, derp.Wrap(err, "service.Source", "Error loading Source", criteria)
	}

	return account, nil
}

// Save adds/updates an Source in the database
func (service Source) Save(account *model.Source, note string) *derp.Error {

	if err := service.session.Save(CollectionSource, account, note); err != nil {
		return derp.Wrap(err, "service.Source", "Error saving Source", account, note)
	}

	return nil
}

// Delete removes an Source from the database (virtual delete)
func (service Source) Delete(account *model.Source, note string) *derp.Error {

	if err := service.session.Delete(CollectionSource, account, note); err != nil {
		return derp.Wrap(err, "service.Source", "Error deleting Source", account, note)
	}

	return nil
}

//// GENERIC FUNCTIONS //////////////////

// NewObject wraps the `New` method as a generic Object
func (service Source) NewObject() data.Object {
	return service.New()
}

// ListObjects wraps the `List` method as a generic Object
func (service Source) ListObjects(criteria expression.Expression, options ...option.Option) (data.Iterator, *derp.Error) {
	return service.List(criteria, options...)
}

// LoadObject wraps the `Load` method as a generic Object
func (service Source) LoadObject(criteria expression.Expression) (data.Object, *derp.Error) {
	return service.Load(criteria)
}

// SaveObject wraps the `Save` method as a generic Object
func (service Source) SaveObject(object data.Object, note string) *derp.Error {

	if object, ok := object.(*model.Source); ok {
		return service.Save(object, note)
	}

	// This should never happen.
	return derp.New(derp.CodeInternalError, "service.Source", "Object is not a model.Source", object, note)
}

// DeleteObject wraps the `Delete` method as a generic Object
func (service Source) DeleteObject(object data.Object, note string) *derp.Error {

	if object, ok := object.(*model.Source); ok {
		return service.Delete(object, note)
	}

	// This should never happen.
	return derp.New(derp.CodeInternalError, "service.Source", "Object is not a model.Source", object, note)
}

// Close cleans up the service and any outstanding connections.
func (service Source) Close() {
	service.session.Close()
}

/// QUERIES //////////////////////////////////

func (service Source) ListSourcesByMethod(method model.SourceMethod) (data.Iterator, *derp.Error) {
	return service.List(expression.New("method", "=", string(method)))
}

//////////////////////////////////////////////

func (service Source) Poll() (*derp.Error, []*derp.Error) {

	var pollErrors []*derp.Error

	object := service.New()

	it, err := service.ListSourcesByMethod(model.SourceMethodPoll)

	if err != nil {
		return derp.Wrap(err, "service.Source.Poll", "Error loading list of sources"), pollErrors
	}

	// Use the stream service to add/remove streams.
	streamService := service.factory.Stream()

	for it.Next(object) {

		adapter, err := source.New(object.Adapter, object.SourceID, object.Config)

		if err != nil {
			pollErrors = append(pollErrors, derp.New(500, "service.Source.Poll", "Error initializing adapter", object))
			continue
		}

		// Load all streams from the adapter.
		streams, err := adapter.Poll()

		if err != nil {
			pollErrors = append(pollErrors, derp.Wrap(err, "service.Source.Poll", "Error retrieving streams from adapter"))
			continue
		}

		// TODO: There HAS to be a more efficient way of diffing streams. Hashes? Batches?
		for _, stream := range streams {
			if err := streamService.SaveUniqueStreamBySourceURL(&stream, "Imported from remote Source"); err != nil {
				pollErrors = append(pollErrors, derp.Wrap(err, "service.Source.Poll", "Error saving unique stream data", stream))
				break
			}
		}
	}

	return nil, pollErrors
}
