package service

import (
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/ghost/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Attachment manages all interactions with the Attachment collection
type Attachment struct {
	collection data.Collection
}

// NewAttachment returns a fully populated Attachment service
func NewAttachment(collection data.Collection) Attachment {
	return Attachment{
		collection: collection,
	}
}

// New creates a newly initialized Attachment that is ready to use
func (service Attachment) New() model.Attachment {
	return model.Attachment{
		AttachmentID: primitive.NewObjectID(),
	}
}

// List returns an iterator containing all of the Attachments who match the provided criteria
func (service Attachment) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(criteria, options...)
}

// Load retrieves an Attachment from the database
func (service Attachment) Load(criteria exp.Expression, result *model.Attachment) error {

	if err := service.collection.Load(criteria, result); err != nil {
		return derp.Wrap(err, "service.Attachment", "Error loading Attachment", criteria)
	}

	return nil
}

// Save adds/updates an Attachment in the database
func (service Attachment) Save(attachment *model.Attachment, note string) error {

	if err := service.collection.Save(attachment, note); err != nil {
		return derp.Wrap(err, "service.Attachment", "Error saving Attachment", attachment, note)
	}

	return nil
}

// Delete removes an Attachment from the database (virtual delete)
func (service Attachment) Delete(attachment *model.Attachment, note string) error {

	if err := service.collection.Delete(attachment, note); err != nil {
		return derp.Wrap(err, "service.Attachment", "Error deleting Attachment", attachment, note)
	}

	return nil
}
