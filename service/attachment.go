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
	factory    *Factory
	collection data.Collection
}

// New creates a newly initialized Attachment that is ready to use
func (service Attachment) New() *model.Attachment {
	return &model.Attachment{
		AttachmentID: primitive.NewObjectID(),
	}
}

// List returns an iterator containing all of the Attachments who match the provided criteria
func (service Attachment) List(criteria expression.Expression, options ...option.Option) (data.Iterator, *derp.Error) {
	return service.collection.List(criteria, options...)
}

// Load retrieves an Attachment from the database
func (service Attachment) Load(criteria expression.Expression) (*model.Attachment, *derp.Error) {

	attachment := service.New()

	if err := service.collection.Load(criteria, attachment); err != nil {
		return nil, derp.Wrap(err, "service.Attachment", "Error loading Attachment", criteria)
	}

	return attachment, nil
}

// Save adds/updates an Attachment in the database
func (service Attachment) Save(attachment *model.Attachment, note string) *derp.Error {

	if err := service.collection.Save(attachment, note); err != nil {
		return derp.Wrap(err, "service.Attachment", "Error saving Attachment", attachment, note)
	}

	return nil
}

// Delete removes an Attachment from the database (virtual delete)
func (service Attachment) Delete(attachment *model.Attachment, note string) *derp.Error {

	if err := service.collection.Delete(attachment, note); err != nil {
		return derp.Wrap(err, "service.Attachment", "Error deleting Attachment", attachment, note)
	}

	return nil
}
