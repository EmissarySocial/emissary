package service

import (
	"github.com/benpate/data"
	"github.com/benpate/data/expression"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CollectionPost is the database collection where Posts are stored
const CollectionPost = "Post"

// Post manosts all interactions with the Post collection
type Post struct {
	factory Factory
	session data.Session
}

// New creates a newly initialized Post that is ready to use
func (service Post) New() *model.Post {
	return &model.Post{
		PostID: primitive.NewObjectID(),
	}
}

// List returns an iterator containing all of the Posts who match the provided criteria
func (service Post) List(criteria expression.Expression, options ...option.Option) (data.Iterator, error) {
	return service.session.List(CollectionPost, criteria, options...)
}

// Load retrieves an Post from the database
func (service Post) Load(criteria expression.Expression) (*model.Post, error) {

	post := service.New()

	if err := service.session.Load(CollectionPost, criteria, post); err != nil {
		return nil, derp.Wrap(err, "service.Post", "Error loading Post", criteria)
	}

	return post, nil
}

// Save adds/updates an Post in the database
func (service Post) Save(post *model.Post, note string) error {

	if err := service.session.Save(CollectionPost, post, note); err != nil {
		return derp.Wrap(err, "service.Post", "Error saving Post", post, note)
	}

	return nil
}

// Delete removes an Post from the database (virtual delete)
func (service Post) Delete(post *model.Post, note string) error {

	if err := service.session.Delete(CollectionPost, post, note); err != nil {
		return derp.Wrap(err, "service.Post", "Error deleting Post", post, note)
	}

	return nil
}

//// GENERIC FUNCTIONS //////////////////

// NewObject wraps the `New` method as a generic Object
func (service Post) NewObject() data.Object {
	return service.New()
}

// ListObjects wraps the `List` method as a generic Object
func (service Post) ListObjects(criteria expression.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

// LoadObject wraps the `Load` method as a generic Object
func (service Post) LoadObject(criteria expression.Expression) (data.Object, error) {
	return service.Load(criteria)
}

// SaveObject wraps the `Save` method as a generic Object
func (service Post) SaveObject(object data.Object, note string) error {

	if object, ok := object.(*model.Post); ok {
		return service.Save(object, note)
	}

	// This should never happen.
	return derp.New(derp.CodeInternalError, "service.Post", "Object is not a model.Post", object, note)
}

// DeleteObject wraps the `Delete` method as a generic Object
func (service Post) DeleteObject(object data.Object, note string) error {

	if object, ok := object.(*model.Post); ok {
		return service.Delete(object, note)
	}

	// This should never happen.
	return derp.New(derp.CodeInternalError, "service.Post", "Object is not a model.Post", object, note)
}

// Close cleans up the service and any outstanding connections.
func (service Post) Close() {
	service.session.Close()
}
