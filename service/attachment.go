package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/mediaserver"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Attachment manages all interactions with the Attachment collection
type Attachment struct {
	collection  data.Collection
	mediaServer mediaserver.MediaServer
}

// NewAttachment returns a fully populated Attachment service
func NewAttachment() Attachment {
	return Attachment{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Attachment) Refresh(collection data.Collection, mediaServer mediaserver.MediaServer) {
	service.collection = collection
	service.mediaServer = mediaServer
}

// Close stops any background processes controlled by this service
func (service *Attachment) Close() {

}

/******************************************
 * Common Data Methods
 ******************************************/

// New creates a newly initialized Attachment that is ready to use
func (service *Attachment) New() model.Attachment {
	return model.NewAttachment("", primitive.NilObjectID)
}

// List returns an iterator containing all of the Attachments who match the provided criteria
func (service *Attachment) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(notDeleted(criteria), options...)
}

func (service *Attachment) Query(criteria exp.Expression, options ...option.Option) ([]model.Attachment, error) {

	result := make([]model.Attachment, 0)

	if err := service.collection.Query(&result, notDeleted(criteria), options...); err != nil {
		return result, derp.Wrap(err, "service.Attachment", "Error querying Attachments", criteria, options)
	}

	return result, nil
}

// Load retrieves an Attachment from the database
func (service *Attachment) Load(criteria exp.Expression, result *model.Attachment) error {

	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Attachment", "Error loading Attachment", criteria)
	}

	return nil
}

// Save adds/updates an Attachment in the database
func (service *Attachment) Save(attachment *model.Attachment, note string) error {

	// Clean the value before saving
	if err := service.Schema().Clean(attachment); err != nil {
		return derp.Wrap(err, "service.Attachment.Save", "Error cleaning Attachment", attachment)
	}

	if err := service.collection.Save(attachment, note); err != nil {
		return derp.Wrap(err, "service.Attachment", "Error saving Attachment", attachment, note)
	}

	return nil
}

// Delete removes an Attachment from the database (virtual delete)
func (service *Attachment) Delete(attachment *model.Attachment, note string) error {

	// Delete uploaded files from MediaServer
	// nolint:errcheck
	if err := service.mediaServer.Delete(attachment.AttachmentID.Hex()); err != nil {
		derp.Report(derp.Wrap(err, "service.Attachment", "Error deleting attached files", attachment))
		// Fail loudly, but do not stop.
	}

	// Delete Attachment record last.
	if err := service.collection.Delete(attachment, note); err != nil {
		return derp.Wrap(err, "service.Attachment", "Error deleting Attachment", attachment, note)
	}

	return nil
}

func (service *Attachment) Schema() schema.Schema {
	return schema.New(model.AttachmentSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *Attachment) QueryByObjectID(objectType string, objectID primitive.ObjectID) ([]model.Attachment, error) {
	return service.Query(
		exp.Equal("objectType", objectType).
			AndEqual("objectId", objectID),
		option.SortAsc("rank"))
}

func (service *Attachment) LoadFirstByObjectID(objectType string, objectID primitive.ObjectID) (model.Attachment, error) {

	attachments, err := service.Query(
		exp.Equal("objectType", objectType).
			AndEqual("objectId", objectID),
		option.SortAsc("rank"), option.FirstRow())

	if err != nil {
		return model.Attachment{}, derp.Wrap(err, "service.Attachment.LoadFirstByObjectID", "Error loading first attachment", objectType, objectID)
	}

	for _, attachment := range attachments {
		return attachment, err
	}

	return model.Attachment{}, derp.Wrap(err, "service.Attachment", "No attachments found", objectType, objectID)
}

func (service *Attachment) LoadByID(objectType string, objectID primitive.ObjectID, attachmentID primitive.ObjectID) (model.Attachment, error) {
	var result model.Attachment
	criteria := exp.Equal("_id", attachmentID).
		AndEqual("objectType", objectType).
		AndEqual("objectId", objectID)
	err := service.Load(criteria, &result)
	return result, err
}

func (service *Attachment) DeleteByID(objectType string, objectID primitive.ObjectID, attachmentID primitive.ObjectID) error {

	const location = "service.Attachment.DeleteByID"

	attachment, err := service.LoadByID(objectType, objectID, attachmentID)

	if err != nil {
		return derp.Wrap(err, location, "Error loading attachment")
	}

	// Delete the attachment
	if err := service.Delete(&attachment, "Deleted"); err != nil {
		return derp.Wrap(err, location, "Error deleting attachment")
	}

	return nil
}

// DeleteByStream removes all attachments from the provided stream (virtual delete)
func (service *Attachment) DeleteAll(objectType string, objectID primitive.ObjectID, note string) error {

	attachments, err := service.QueryByObjectID(objectType, objectID)

	if err != nil {
		return derp.Wrap(err, "service.Attachment.DeleteByStream", "Error listing attachments", objectID)
	}

	for _, attachment := range attachments {
		if err := service.Delete(&attachment, note); err != nil {
			derp.Report(derp.Wrap(err, "service.Attachment.DeleteByStream", "Error deleting child stream", attachment))
			// Fail loudly, but do not stop.
		}
	}

	return nil
}
