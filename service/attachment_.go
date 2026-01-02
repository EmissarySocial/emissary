package service

import (
	"iter"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/mediaserver"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Attachment manages all interactions with the Attachment collection
type Attachment struct {
	host              string
	importItemService *ImportItem
	mediaServer       mediaserver.MediaServer
}

// NewAttachment returns a fully populated Attachment service
func NewAttachment() Attachment {
	return Attachment{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Attachment) Refresh(factory *Factory) {
	service.host = factory.Host()
	service.importItemService = factory.ImportItem()
	service.mediaServer = factory.MediaServer()
}

// Close stops any background processes controlled by this service
func (service *Attachment) Close() {

}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *Attachment) collection(session data.Session) data.Collection {
	return session.Collection("Attachment")
}

// New creates a newly initialized Attachment that is ready to use
func (service *Attachment) New() model.Attachment {
	return model.NewAttachment("", primitive.NilObjectID)
}

// Count returns the number of records that match the provided criteria
func (service *Attachment) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// List returns an iterator containing all of the Attachments who match the provided criteria
func (service *Attachment) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the Streams that match the provided criteria
func (service *Attachment) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.Attachment], error) {

	const location = "service.Attachment.Range"

	iter, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to create iterator", criteria)
	}

	return RangeFunc(iter, model.NewEmptyAttachment), nil
}

func (service *Attachment) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.Attachment, error) {

	const location = "service.Attachment.Query"

	result := make([]model.Attachment, 0)

	if err := service.collection(session).Query(&result, notDeleted(criteria), options...); err != nil {
		return result, derp.Wrap(err, location, "Unable to query Attachments", criteria, options)
	}

	return result, nil
}

// Load retrieves an Attachment from the database
func (service *Attachment) Load(session data.Session, criteria exp.Expression, result *model.Attachment) error {

	const location = "service.Attachment.Load"

	if err := service.collection(session).Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, location, "Unable to load Attachment", criteria)
	}

	return nil
}

// Save adds/updates an Attachment in the database
func (service *Attachment) Save(session data.Session, attachment *model.Attachment, note string) error {

	const location = "service.Attachment.Save"

	// Validate the value before saving
	if err := service.Schema().Validate(attachment); err != nil {
		return derp.Wrap(err, location, "Unable to validate Attachment", attachment)
	}

	// Calculate the URL
	attachment.URL = attachment.CalcURL(service.host)

	// Save the record to the database
	if err := service.collection(session).Save(attachment, note); err != nil {
		return derp.Wrap(err, location, "Unable to save Attachment", attachment, note)
	}

	return nil
}

// Delete removes an Attachment from the database (virtual delete)
func (service *Attachment) Delete(session data.Session, attachment *model.Attachment, note string) error {

	const location = "service.Attachment.Delete"

	// Delete uploaded files from MediaServer
	if err := service.mediaServer.Delete(attachment.AttachmentID.Hex()); err != nil {
		derp.Report(derp.Wrap(err, location, "Unable to delete attached files", attachment))
		// Fail loudly, but do not stop.
	}

	// Delete Attachment record last.
	if err := service.collection(session).Delete(attachment, note); err != nil {
		return derp.Wrap(err, location, "Unable to delete Attachment", attachment, note)
	}

	return nil
}

/******************************************
 * Special Case Methods
 ******************************************/

// QueryIDOnly returns a slice of IDOnly records that match the provided criteria
func (service *Attachment) QueryIDOnly(session data.Session, criteria exp.Expression, options ...option.Option) (sliceof.Object[model.IDOnly], error) {
	result := make([]model.IDOnly, 0)
	options = append(options, option.Fields("_id"))
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// HardDeleteByID removes a specific Attachment record, without applying any additional business rules
func (service *Attachment) HardDeleteByID(session data.Session, userID primitive.ObjectID, attachmentID primitive.ObjectID) error {

	const location = "service.Attachment.HardDeleteByID"

	criteria := exp.Equal("userId", userID).AndEqual("_id", attachmentID)

	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Unable to delete Attachment", "userID: "+userID.Hex(), "attachmentID: "+attachmentID.Hex())
	}

	return nil
}

/******************************************
 * Generic Data Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Attachment) ObjectType() string {
	return "Attachment"
}

// New returns a fully initialized model.Attachment as a data.Object.
func (service *Attachment) ObjectNew() data.Object {
	result := model.NewAttachment("", primitive.NilObjectID)
	return &result
}

// ObjectID retrieves the AttachmentID from the provided object
func (service *Attachment) ObjectID(object data.Object) primitive.ObjectID {

	if attachment, ok := object.(*model.Attachment); ok {
		return attachment.AttachmentID
	}

	return primitive.NilObjectID
}

// ObjectQuery returns a slice of Attachments that match the provided criteria (generically)
func (service *Attachment) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

// ObjectLoad retrieves an Attachment from the database (generically)
func (service *Attachment) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewAttachment("", primitive.NilObjectID)
	err := service.Load(session, criteria, &result)
	return &result, err
}

// ObjectSave adds/updates an Attachment in the database (generically)
func (service *Attachment) ObjectSave(session data.Session, object data.Object, note string) error {

	if attachment, ok := object.(*model.Attachment); ok {
		return service.Save(session, attachment, note)
	}
	return derp.Internal("service.Attachment.ObjectSave", "Invalid object type", object)
}

// ObjectDelete removes an Attachment from the database (generically)
func (service *Attachment) ObjectDelete(session data.Session, object data.Object, note string) error {
	if attachment, ok := object.(*model.Attachment); ok {
		return service.Delete(session, attachment, note)
	}
	return derp.Internal("service.Attachment.ObjectDelete", "Invalid object type", object)
}

// ObjectUserCan returns true if the current user has permission to perform the requested action on the provided Attachment
func (service *Attachment) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.Unauthorized("service.Attachment.ObjectUserCan", "Not Authorized")
}

// Schema returns the schema that this service uses to validate Attachments
func (service *Attachment) Schema() schema.Schema {
	return schema.New(model.AttachmentSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

// QueryByObjectID returns all Attachments that match the provided object (type and ID)
func (service *Attachment) QueryByObjectID(session data.Session, objectType string, objectID primitive.ObjectID) ([]model.Attachment, error) {
	return service.Query(
		session,
		exp.Equal("objectType", objectType).
			AndEqual("objectId", objectID),
		option.SortAsc("rank"))
}

// QueryByCategory returns all Attachments that match the provided object (type and ID).  If the "category"
// parameter is provided, then only Attachments with that category will be returned.
func (service *Attachment) QueryByCategory(session data.Session, objectType string, objectID primitive.ObjectID, category string) ([]model.Attachment, error) {

	criteria := exp.Equal("objectType", objectType).
		AndEqual("objectId", objectID)

	if category != "" {
		criteria = criteria.AndEqual("category", category)
	}

	return service.Query(session, criteria, option.SortAsc("rank"))
}

func (service *Attachment) LoadFirstByCategory(session data.Session, objectType string, objectID primitive.ObjectID, categories []string) (model.Attachment, error) {

	const location = "service.Attachment.LoadFirstByCategory"

	attachments, err := service.Query(
		session,
		exp.Equal("objectType", objectType).
			AndEqual("objectId", objectID).
			AndIn("category", categories),
		option.SortAsc("rank"), option.FirstRow())

	if err != nil {
		return model.Attachment{}, derp.Wrap(err, location, "Unable to load first attachment", objectType, objectID)
	}

	for _, attachment := range attachments {
		return attachment, err
	}

	return model.Attachment{}, derp.NotFound(location, "No attachments found", objectType, objectID)
}

func (service *Attachment) LoadFirstByObjectID(session data.Session, objectType string, objectID primitive.ObjectID) (model.Attachment, error) {

	const location = "service.Attachment.LoadFirstByObjectID"

	attachments, err := service.Query(
		session,
		exp.Equal("objectType", objectType).
			AndEqual("objectId", objectID),
		option.SortAsc("rank"), option.FirstRow())

	if err != nil {
		return model.Attachment{}, derp.Wrap(err, location, "Unable to load first attachment", objectType, objectID)
	}

	for _, attachment := range attachments {
		return attachment, err
	}

	return model.Attachment{}, derp.NotFound(location, "No attachments found", objectType, objectID)
}

func (service *Attachment) LoadByID(session data.Session, objectType string, objectID primitive.ObjectID, attachmentID primitive.ObjectID, result *model.Attachment) error {

	criteria := exp.Equal("_id", attachmentID).
		AndEqual("objectType", objectType).
		AndEqual("objectId", objectID)

	if err := service.Load(session, criteria, result); err != nil {
		return derp.Wrap(err, "service.Attachment.LoadByID", "Unable to load attachment", "objectType: "+objectType, "objectID: "+objectID.Hex(), "attachmentID: "+attachmentID.Hex())
	}

	return nil
}

func (service *Attachment) LoadByToken(session data.Session, objectType string, objectID primitive.ObjectID, token string, result *model.Attachment) error {

	const location = "service.Attachment.LoadByToken"

	attachmentID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.NotFound(location, "AttachmentID must be a valid ObjectID", token)
	}

	return service.LoadByID(session, objectType, objectID, attachmentID, result)
}

func (service *Attachment) DeleteByID(session data.Session, objectType string, objectID primitive.ObjectID, attachmentID primitive.ObjectID, note string) error {

	const location = "service.Attachment.DeleteByID"

	// Load the Attachment from the database
	attachment := model.NewAttachment(objectType, objectID)
	if err := service.LoadByID(session, objectType, objectID, attachmentID, &attachment); err != nil {
		return derp.Wrap(err, location, "Unable to load attachment")
	}

	// Delete the attachment
	if err := service.Delete(session, &attachment, note); err != nil {
		return derp.Wrap(err, location, "Unable to delete attachment")
	}

	// Success.
	return nil
}

// DeleteByCriteria removes all attachments from the provided object/type within a criteria expression (virtual delete)
func (service *Attachment) DeleteByCriteria(session data.Session, objectType string, objectID primitive.ObjectID, criteria exp.Expression, note string) error {

	const location = "service.Attachment.DeleteByStream"

	// Append the object/type criteria to the provided criteria
	criteria = criteria.
		AndEqual("objectType", objectType).
		AndEqual("objectId", objectID)

	// Query for all attachments that match the criteria
	attachments, err := service.Query(session, criteria)

	if err != nil {
		return derp.Wrap(err, location, "Unable to list attachments", objectID)
	}

	// Delete each attachment individually
	for _, attachment := range attachments {
		if err := service.Delete(session, &attachment, note); err != nil {
			derp.Report(derp.Wrap(err, location, "Unable to delete child stream", attachment))
		}
	}

	// Bravo!!
	return nil
}

// DeleteAll removes all attachments from the provided object/type (virtual delete)
func (service *Attachment) DeleteAll(session data.Session, objectType string, objectID primitive.ObjectID, note string) error {
	return service.DeleteByCriteria(session, objectType, objectID, exp.All(), note)
}

/******************************************
 * Custom Behaviors
 ******************************************/

// MakeRoom removes attachments (by object and category) that exceed the provided maximum.
func (service *Attachment) MakeRoom(session data.Session, objectType string, objectID primitive.ObjectID, category string, action string, maximum int, addCount int) error {

	const location = "service.Attachment.MakeRoom"

	log.Trace().Str("objectType", objectType).Str("objectID", objectID.Hex()).Str("category", category).Int("maximum", maximum).Int("addCount", addCount).Msg("MakeRoom")

	// If the maximum is zero, then there's no limit to the number of attachments.
	if maximum == 0 {
		return nil
	}

	// Find the existing Attachments
	attachments, err := service.QueryByCategory(session, objectType, objectID, category)

	if err != nil {
		return derp.Wrap(err, location, "Unable to find existing attachments", objectType, objectID)
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

		log.Trace().Str("attachmentID", attachment.AttachmentID.Hex()).Msg("Removing attachment")
		if err := service.Delete(session, &attachment, "Deleted"); err != nil {
			return derp.Wrap(err, location, "Unable to remove attachment")
		}
	}

	return nil
}
