package service

import (
	"github.com/benpate/data"
	"github.com/benpate/data/expression"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CollectionComment is the database collection where Comments are stored
const CollectionComment = "Comment"

// Comment manages all interactions with the Comment collection
type Comment struct {
	factory *Factory
	session data.Session
}

// New creates a newly initialized Comment that is ready to use
func (service Comment) New() *model.Comment {
	return &model.Comment{
		CommentID: primitive.NewObjectID(),
	}
}

// List returns an iterator containing all of the Comments who match the provided criteria
func (service Comment) List(criteria expression.Expression, options ...option.Option) (data.Iterator, *derp.Error) {
	return service.session.List(CollectionComment, criteria, options...)
}

// Load retrieves an Comment from the database
func (service Comment) Load(criteria expression.Expression) (*model.Comment, *derp.Error) {

	comment := service.New()

	if err := service.session.Load(CollectionComment, criteria, comment); err != nil {
		return nil, derp.Wrap(err, "service.Comment", "Error loading Comment", criteria)
	}

	return comment, nil
}

// Save adds/updates an Comment in the database
func (service Comment) Save(comment *model.Comment, note string) *derp.Error {

	if err := service.session.Save(CollectionComment, comment, note); err != nil {
		return derp.Wrap(err, "service.Comment", "Error saving Comment", comment, note)
	}

	return nil
}

// Delete removes an Comment from the database (virtual delete)
func (service Comment) Delete(comment *model.Comment, note string) *derp.Error {

	if err := service.session.Delete(CollectionComment, comment, note); err != nil {
		return derp.Wrap(err, "service.Comment", "Error deleting Comment", comment, note)
	}

	return nil
}

//// GENERIC FUNCTIONS //////////////////

// NewObject wraps the `New` method as a generic Object
func (service Comment) NewObject() data.Object {
	return service.New()
}

// ListObjects wraps the `List` method as a generic Object
func (service Comment) ListObjects(criteria expression.Expression, options ...option.Option) (data.Iterator, *derp.Error) {
	return service.List(criteria, options...)
}

// LoadObject wraps the `Load` method as a generic Object
func (service Comment) LoadObject(criteria expression.Expression) (data.Object, *derp.Error) {
	return service.Load(criteria)
}

// SaveObject wraps the `Save` method as a generic Object
func (service Comment) SaveObject(object data.Object, note string) *derp.Error {

	if object, ok := object.(*model.Comment); ok {
		return service.Save(object, note)
	}

	// This should never happen.
	return derp.New(derp.CodeInternalError, "service.Comment", "Object is not a model.Comment", object, note)
}

// DeleteObject wraps the `Delete` method as a generic Object
func (service Comment) DeleteObject(object data.Object, note string) *derp.Error {

	if object, ok := object.(*model.Comment); ok {
		return service.Delete(object, note)
	}

	// This should never happen.
	return derp.New(derp.CodeInternalError, "service.Comment", "Object is not a model.Comment", object, note)
}

// Close cleans up the service and any outstanding connections.
func (service Comment) Close() {
	service.session.Close()
}
