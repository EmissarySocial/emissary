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
	host        string
}

// NewAttachment returns a fully populated Attachment service
func NewAttachment() Attachment {
	return Attachment{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Attachment) Refresh(collection data.Collection, mediaServer mediaserver.MediaServer, host string) {
	service.collection = collection
	service.mediaServer = mediaServer
	service.host = host
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
	return service.collection.Iterator(notDeleted(criteria), options...)
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

	// Validate the value before saving
	if err := service.Schema().Validate(attachment); err != nil {
		return derp.Wrap(err, "service.Attachment.Save", "Error validating Attachment", attachment)
	}

	// Calculate the URL
	attachment.SetURL(service.host)

	// Save the record to the database
	if err := service.collection.Save(attachment, note); err != nil {
		return derp.Wrap(err, "service.Attachment", "Error saving Attachment", attachment, note)
	}

	return nil
}

// Delete removes an Attachment from the database (virtual delete)
func (service *Attachment) Delete(attachment *model.Attachment, note string) error {

	// Delete uploaded files from MediaServer
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

// QueryByObjectID returns all Attachments that match the provided object (type and ID)
func (service *Attachment) QueryByObjectID(objectType string, objectID primitive.ObjectID) ([]model.Attachment, error) {
	return service.Query(
		exp.Equal("objectType", objectType).
			AndEqual("objectId", objectID),
		option.SortAsc("rank"))
}

// QueryByCategory returns all Attachments that match the provided object (type and ID).  If the "category"
// parameter is provided, then only Attachments with that category will be returned.
func (service *Attachment) QueryByCategory(objectType string, objectID primitive.ObjectID, category string) ([]model.Attachment, error) {

	criteria := exp.Equal("objectType", objectType).
		AndEqual("objectId", objectID)

	if category != "" {
		criteria = criteria.AndEqual("category", category)
	}

	return service.Query(criteria, option.SortAsc("rank"))
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

func (service *Attachment) LoadByID(objectType string, objectID primitive.ObjectID, attachmentID primitive.ObjectID, result *model.Attachment) error {

	criteria := exp.Equal("_id", attachmentID).
		AndEqual("objectType", objectType).
		AndEqual("objectId", objectID)

	if err := service.Load(criteria, result); err != nil {
		return derp.Wrap(err, "service.Attachment.LoadByID", "Error loading attachment", objectType, objectID, attachmentID)
	}

	return nil
}

func (service *Attachment) DeleteByID(objectType string, objectID primitive.ObjectID, attachmentID primitive.ObjectID, note string) error {

	const location = "service.Attachment.DeleteByID"

	// Load the Attachment from the database
	attachment := model.NewAttachment(objectType, objectID)
	if err := service.LoadByID(objectType, objectID, attachmentID, &attachment); err != nil {
		return derp.Wrap(err, location, "Error loading attachment")
	}

	// Delete the attachment
	if err := service.Delete(&attachment, note); err != nil {
		return derp.Wrap(err, location, "Error deleting attachment")
	}

	// Success.
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
		}
	}

	return nil
}

/******************************************
 * Custom Behaviors
 ******************************************/

// MakeRoom removes attachments (by object and category) that exceed the provided maximum.
func (service *Attachment) MakeRoom(objectType string, objectID primitive.ObjectID, category string, action string, maximum int, addCount int) error {

	const location = "service.Attachment.MakeRoom"

	// If the maximum is zero, then there's no limit to the number of attachments.
	if maximum == 0 {
		return nil
	}

	// Find the existing Attachments
	attachments, err := service.QueryByCategory(objectType, objectID, category)

	if err != nil {
		return derp.Wrap(err, location, "Error finding existing attachments", objectType, objectID)
	}

	currentCount := len(attachments)

	// If there are no Attachments, then there's no "room" to make.
	if currentCount == 0 {
		return nil
	}

	// Let's figure out how many attachments to delete from the front of the results.
	var removeCount int

	switch action {

	// If "replace" then remove ALL existing attachments
	case "replace":
		removeCount = currentCount

	// Default case is "append".  Only remove the attachments that overflow the maximum
	default:
		removeCount = currentCount + addCount - maximum
	}

	// If there's nothing to do, then there's nothing to do.
	if removeCount <= 0 {
		return nil
	}

	// Delete overflowing attachments (starting with the beginning of the result slice)
	for index := 0; index < removeCount; index++ {
		attachment := attachments[index]

		if err := service.Delete(&attachment, "Deleted"); err != nil {
			return derp.Wrap(err, location, "Error removing attachment")
		}
	}

	return nil
}
