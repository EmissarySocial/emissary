package service

import (
	"github.com/benpate/data"
	"github.com/benpate/data/expression"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CollectionAttachment is the database collection where Attachments are stored
const CollectionAttachment = "Attachment"

// Attachment manages all interactions with the Attachment collection
type Attachment struct {
	factory Factory
	session data.Session
}

// New creates a newly initialized Attachment that is ready to use
func (service Attachment) New() *model.Attachment {
	return &model.Attachment{
		AttachmentID: primitive.NewObjectID(),
	}
}

// List returns an iterator containing all of the Attachments who match the provided criteria
func (service Attachment) List(criteria expression.Expression, options ...option.Option) (data.Iterator, *derp.Error) {
	return service.session.List(CollectionAttachment, criteria, options...)
}

// Load retrieves an Attachment from the database
func (service Attachment) Load(criteria expression.Expression) (*model.Attachment, *derp.Error) {

	attachment := service.New()

	if err := service.session.Load(CollectionAttachment, criteria, attachment); err != nil {
		return nil, derp.Wrap(err, "service.Attachment", "Error loading Attachment", criteria)
	}

	return attachment, nil
}

// Save adds/updates an Attachment in the database
func (service Attachment) Save(attachment *model.Attachment, note string) *derp.Error {

	if err := service.session.Save(CollectionAttachment, attachment, note); err != nil {
		return derp.Wrap(err, "service.Attachment", "Error saving Attachment", attachment, note)
	}

	return nil
}

// Delete removes an Attachment from the database (virtual delete)
func (service Attachment) Delete(attachment *model.Attachment, note string) *derp.Error {

	if err := service.session.Delete(CollectionAttachment, attachment, note); err != nil {
		return derp.Wrap(err, "service.Attachment", "Error deleting Attachment", attachment, note)
	}

	return nil
}

//// GENERIC FUNCTIONS //////////////////

// NewObject wraps the `New` method as a generic Object
func (service Attachment) NewObject() data.Object {
	return service.New()
}

// ListObjects wraps the `List` method as a generic Object
func (service Attachment) ListObjects(criteria expression.Expression, options ...option.Option) (data.Iterator, *derp.Error) {
	return service.List(criteria, options...)
}

// LoadObject wraps the `Load` method as a generic Object
func (service Attachment) LoadObject(criteria expression.Expression) (data.Object, *derp.Error) {
	return service.Load(criteria)
}

// SaveObject wraps the `Save` method as a generic Object
func (service Attachment) SaveObject(object data.Object, note string) *derp.Error {

	if object, ok := object.(*model.Attachment); ok {
		return service.Save(object, note)
	}

	// This should never happen.
	return derp.New(derp.CodeInternalError, "service.Attachment", "Object is not a model.Attachment", object, note)
}

// DeleteObject wraps the `Delete` method as a generic Object
func (service Attachment) DeleteObject(object data.Object, note string) *derp.Error {

	if object, ok := object.(*model.Attachment); ok {
		return service.Delete(object, note)
	}

	// This should never happen.
	return derp.New(derp.CodeInternalError, "service.Attachment", "Object is not a model.Attachment", object, note)
}

// Close cleans up the service and any outstanding connections.
func (service Attachment) Close() {
	service.session.Close()
}
