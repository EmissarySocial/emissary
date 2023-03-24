package service

import (
	"context"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Stream manages all interactions with the Stream collection
type Stream struct {
	collection          data.Collection
	templateService     *Template
	draftService        *StreamDraft
	attachmentService   *Attachment
	host                string
	streamUpdateChannel chan<- model.Stream
}

// NewStream returns a fully populated Stream service.
func NewStream(collection data.Collection, templateService *Template, attachmentService *Attachment, host string, streamUpdateChannel chan model.Stream) Stream {

	service := Stream{
		templateService:     templateService,
		attachmentService:   attachmentService,
		host:                host,
		streamUpdateChannel: streamUpdateChannel,
	}

	service.Refresh(host, collection, nil)

	return service
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Stream) Refresh(host string, collection data.Collection, draftService *StreamDraft) {
	service.host = host
	service.collection = collection
	service.draftService = draftService
}

// Close stops any background processes controlled by this service
func (service *Stream) Close() {

}

/******************************************
 * Common Data Methods
 ******************************************/

// New returns a new stream that uses the named template.
func (service *Stream) New(navigationID string, parentID primitive.ObjectID, templateID string) (model.Stream, model.Template, error) {

	const location = "service.Stream.New"

	template, err := service.templateService.Load(templateID)

	if err != nil {
		return model.Stream{}, template, derp.Wrap(err, location, "Invalid template", templateID)
	}

	result := model.NewStream()
	result.TemplateID = templateID
	result.NavigationID = navigationID
	result.ParentID = parentID

	// TODO: HIGH: Use stream Template schema to set default values in the new stream.

	return result, template, nil
}

// Query returns an slice containing all of the Streams that match the provided criteria
func (service *Stream) Query(criteria exp.Expression, options ...option.Option) ([]model.Stream, error) {
	result := make([]model.Stream, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)
	return result, err
}

// QuerySummary returns an slice containing StreamSummaries for all of the Streams that match the provided criteria
func (service *Stream) QuerySummary(criteria exp.Expression, options ...option.Option) ([]model.StreamSummary, error) {
	result := make([]model.StreamSummary, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator containing all of the Streams that match the provided criteria
func (service *Stream) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(notDeleted(criteria), options...)
}

// Load retrieves an Stream from the database
func (service *Stream) Load(criteria exp.Expression, stream *model.Stream) error {

	if err := service.collection.Load(notDeleted(criteria), stream); err != nil {
		return derp.Wrap(err, "service.Stream", "Error loading Stream", criteria)
	}

	return nil
}

// Save adds/updates an Stream in the database
func (service *Stream) Save(stream *model.Stream, note string) error {

	const location = "service.Stream"

	template, err := service.templateService.Load(stream.TemplateID)

	if err != nil {
		return derp.Wrap(err, location, "Invalid Template", stream.TemplateID)
	}

	// Copy default values from the Template
	stream.SocialRole = template.SocialRole
	stream.Document.URL = service.host + "/" + stream.StreamID.Hex()

	// RULE: Calculate "defaultAllow" groups for this stream.
	defaultTemplate := template.Default()
	defaultRoles := defaultTemplate.AllowedRoles(stream.StateID)
	stream.DefaultAllow = stream.PermissionGroups(defaultRoles...)

	// RULE: Calculate rank
	if stream.Rank == 0 {
		maxRank, err := service.MaxRank(context.TODO(), stream.ParentID)

		if err != nil {
			return derp.Wrap(err, location, "Error calculating max rank")
		}
		stream.Rank = maxRank
	}

	// RULE: Default Token
	if stream.Token == "" {
		stream.Token = stream.StreamID.Hex()
	}

	// Clean the value (using the global stream schema) before saving
	if err := service.Schema().Clean(stream); err != nil {
		return derp.Wrap(err, "service.Stream.Save", "Error cleaning Stream using StreamSchema", stream)
	}

	// Clean the value (using the template-specific schema) before saving
	if err := template.Schema.Clean(stream); err != nil {
		return derp.Wrap(err, "service.Stream.Save", "Error cleaning Stream using TemplateSchema", stream)
	}

	// RULE: If this stream does not have ParentIDs, then calculate them now.
	if len(stream.ParentIDs) == 0 {
		if err := service.CalcParentIDs(stream); err != nil {
			return derp.Wrap(err, location, "Error calculating parent IDs", stream)
		}
	}

	// RULE: Calculate the depth of this stream
	stream.Depth = len(stream.ParentIDs)

	// Try to save the Stream to the database
	if err := service.collection.Save(stream, note); err != nil {
		return derp.Wrap(err, location, "Error saving Stream", stream, note)
	}

	// NON-BLOCKING: Notify other processes on this server that the stream has been updated
	go func() {
		service.streamUpdateChannel <- *stream
	}()

	// One milisecond delay prevents overlapping stream.CreateDates.  Deal with it.
	// TODO: There has to be a better way than this...
	time.Sleep(1 * time.Millisecond)

	return nil
}

// Delete removes an Stream from the database (virtual delete)
func (service *Stream) Delete(stream *model.Stream, note string) error {

	// Delete this Stream
	if err := service.collection.Delete(stream, note); err != nil {
		return derp.Wrap(err, "service.Stream.Delete", "Error deleting Stream", stream, note)
	}

	// Delete related records -- this can happen in the background
	go func() {

		// RULE: Delete all related Children
		if err := service.DeleteByParent(stream.StreamID, note); err != nil {
			derp.Report(derp.Wrap(err, "service.Stream.Delete", "Error deleting child streams", stream, note))
		}

		// RULE: Delete all related Attachments
		if err := service.attachmentService.DeleteAll(model.AttachmentTypeStream, stream.StreamID, note); err != nil {
			derp.Report(derp.Wrap(err, "service.Stream.Delete", "Error deleting attachments", stream, note))
		}

		// RULE: Delete all related Drafts
		if err := service.draftService.Delete(stream, note); err != nil {
			derp.Report(derp.Wrap(err, "service.Stream.Delete", "Error deleting drafts", stream, note))
		}
	}()

	// Bueno!!
	return nil
}

// DeleteMany removes all child streams from the provided stream (virtual delete)
func (service *Stream) DeleteMany(criteria exp.Expression, note string) error {

	it, err := service.List(notDeleted(criteria))

	if err != nil {
		return derp.Wrap(err, "service.Stream.Delete", "Error listing streams to delete", criteria)
	}

	stream := model.NewStream()

	for it.Next(&stream) {
		if err := service.Delete(&stream, note); err != nil {
			return derp.Wrap(err, "service.Stream.Delete", "Error deleting stream", stream)
		}
		stream = model.NewStream()
	}

	return nil
}

/******************************************
 * Generic Data Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Stream) ObjectType() string {
	return "Stream"
}

// New returns a fully initialized model.Stream as a data.Object.
func (service *Stream) ObjectNew() data.Object {
	result := model.NewStream()
	return &result
}

func (service *Stream) ObjectID(object data.Object) primitive.ObjectID {

	if stream, ok := object.(*model.Stream); ok {
		return stream.StreamID
	}

	return primitive.NilObjectID
}

func (service *Stream) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *Stream) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

func (service *Stream) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewStream()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Stream) ObjectSave(object data.Object, note string) error {

	if stream, ok := object.(*model.Stream); ok {
		return service.Save(stream, note)
	}
	return derp.NewInternalError("service.Stream.ObjectSave", "Invalid object type", object)
}

func (service *Stream) ObjectDelete(object data.Object, note string) error {
	if stream, ok := object.(*model.Stream); ok {
		return service.Delete(stream, note)
	}
	return derp.NewInternalError("service.Stream.ObjectDelete", "Invalid object type", object)
}

func (service *Stream) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.NewUnauthorizedError("service.Stream", "Not Authorized")
}

func (service *Stream) Schema() schema.Schema {
	// TODO: HIGH: Implement
	result := schema.New(model.StreamSchema())
	result.ID = "https://emissary.social/schemas/stream"
	return result
}

/******************************************
 * Custom Queries
 ******************************************/

// ListNavigation returns all Streams of type FOLDER at the top of the hierarchy
func (service *Stream) ListNavigation() (data.Iterator, error) {
	return service.List(
		exp.Equal("parentId", primitive.NilObjectID),
		option.SortAsc("rank"),
	)
}

// ListByParent returns all Streams that match a particular parentID
func (service *Stream) ListByParent(parentID primitive.ObjectID) (data.Iterator, error) {
	return service.List(exp.Equal("parentId", parentID))
}

// ListByTemplate returns all `Streams` that use a particular `Template`
func (service *Stream) ListByTemplate(template string) (data.Iterator, error) {
	return service.List(exp.Equal("templateId", template))
}

// LoadByToken returns a single `Stream` that matches a particular `Token`
func (service *Stream) LoadByToken(token string, result *model.Stream) error {

	// If the token looks like an ObjectID, then try Load by ID first.
	if streamID, err := primitive.ObjectIDFromHex(token); err == nil {
		if err := service.LoadByID(streamID, result); err == nil {
			return nil
		}
	}

	// Default to Load by Token
	return service.Load(exp.Equal("token", token), result)
}

// LoadByID returns a single `Stream` that matches the provided streamID
func (service *Stream) LoadByID(streamID primitive.ObjectID, result *model.Stream) error {
	return service.Load(exp.Equal("_id", streamID), result)
}

// LoadByOriginID returns a single `Stream` that matches the provided `Origin.InternalID`
func (service *Stream) LoadByOriginID(originID primitive.ObjectID, result *model.Stream) error {
	return service.Load(exp.Equal("origin.internalId", originID), result)
}

// LoadByProductID returns a single `Stream` with custom data matching the provided `Data.productId`
func (service *Stream) LoadByProductID(productID string, result *model.Stream) error {
	return service.Load(exp.Equal("data.productId", productID), result)
}

// LoadParent returns the Stream that is the parent of the provided Stream
func (service *Stream) LoadParent(stream *model.Stream, parent *model.Stream) error {

	if !stream.HasParent() {
		return derp.NewNotFoundError("service.Stream.LoadParent", "Stream does not have a parent")
	}

	if err := service.LoadByID(stream.ParentID, parent); err != nil {
		return derp.Wrap(err, "service.stream.LoadParent", "Error loading parent", stream)
	}

	return nil
}

// LoadNavigationByID locates a single stream in the top level of the site hierarchy
func (service *Stream) LoadNavigationByID(streamID primitive.ObjectID, result *model.Stream) error {

	criteria := exp.
		Equal("_id", streamID).
		AndEqual("parentId", primitive.NilObjectID)

	return service.Load(criteria, result)
}

func (service *Stream) LoadWithOptions(criteria exp.Expression, result *model.Stream, options ...option.Option) error {

	const location = "service.stream.LoadWithOptions"

	it, err := service.List(notDeleted(criteria), options...)

	if err != nil {
		return derp.Wrap(err, location, "Error getting iterator")
	}

	for it.Next(result) {
		return nil
	}

	return derp.NewNotFoundError(location, "collection is empty")
}

func (service *Stream) LoadFirstSibling(parentID primitive.ObjectID, result *model.Stream) error {
	return service.LoadWithOptions(exp.Equal("parentId", parentID), result, option.SortAsc("rank"))
}

func (service *Stream) LoadPrevSibling(parentID primitive.ObjectID, rank int, result *model.Stream) error {

	if rank == 0 {
		return service.LoadLastSibling(parentID, result)
	}

	criteria := exp.Equal("parentId", parentID).AndLessThan("rank", rank)

	err := service.LoadWithOptions(criteria, result, option.SortDesc("rank"))

	if err == nil {
		return nil
	}

	if derp.NotFound(err) {
		return service.LoadLastSibling(parentID, result)
	}

	return derp.Wrap(err, "service.stream.LoadPreviousSibling", "Error loading Previous Sibling")
}

func (service *Stream) LoadNextSibling(parentID primitive.ObjectID, rank int, result *model.Stream) error {

	criteria := exp.Equal("parentId", parentID).AndGreaterThan("rank", rank)

	err := service.LoadWithOptions(criteria, result, option.SortAsc("rank"))

	if err == nil {
		return nil
	}

	if derp.NotFound(err) {
		return service.LoadFirstSibling(parentID, result)
	}

	return derp.Wrap(err, "service.stream.LoadNextSibling", "Error loading Next Sibling")
}

func (service *Stream) LoadLastSibling(parentID primitive.ObjectID, result *model.Stream) error {
	return service.LoadWithOptions(exp.Equal("parentId", parentID), result, option.SortDesc("rank"))
}

func (service *Stream) LoadFirstAttachment(streamID primitive.ObjectID) (model.Attachment, error) {
	return service.attachmentService.LoadFirstByObjectID(model.AttachmentTypeStream, streamID)
}

// Count returns the number of (non-deleted) records in the Stream collection
func (service *Stream) Count(ctx context.Context, criteria exp.Expression) (int, error) {
	return queries.CountRecords(ctx, service.collection, notDeleted(criteria))
}

// MaxRank returns the maximum rank of all children of a stream
func (service *Stream) MaxRank(ctx context.Context, parentID primitive.ObjectID) (int, error) {
	return queries.MaxRank(ctx, service.collection, parentID)
}

/******************************************
 * Outbox Queries (may move to separate service later)
 ******************************************/

func (service *Stream) Outbox(ownerID primitive.ObjectID, criteria exp.Expression, options ...option.Option) ([]model.StreamSummary, error) {
	criteria = criteria.AndEqual("ownerId", ownerID)
	return service.QuerySummary(criteria, options...)
}

/******************************************
 * Custom Actions
 ******************************************/

func (service *Stream) DeleteByParent(parentID primitive.ObjectID, note string) error {
	return service.DeleteMany(exp.Equal("parentId", parentID), note)
}

// Delete RelatedDuplicate hard deletes any inbox/outbox streams that point to the same original.
func (service *Stream) DeleteRelatedDuplicate(parentID primitive.ObjectID, originalStreamID primitive.ObjectID) error {

	criteria := exp.Equal("parentId", parentID).AndEqual("data.originalStreamId", originalStreamID)

	if err := service.collection.HardDelete(criteria); err != nil {
		return derp.Wrap(err, "service.Stream.DeleteRelatedDuplicate", "Error deleting related duplicate")
	}

	return nil
}

// RestoreDeleted un-deletes all soft-deleted records underneath a common ancestor.
func (service *Stream) RestoreDeleted(ancestorID primitive.ObjectID) error {

	const location = "service.Stream.RestoreDeleted"

	// Try to list all deleted descendents
	criteria := exp.Equal("parentIds", ancestorID).AndGreaterThan("journal.deleteDate", 0)
	iterator, err := service.collection.List(criteria)

	if err != nil {
		return derp.Wrap(err, location, "Error listing soft-deleted streams")
	}

	// Iterate through all descendents and UnDelete
	stream := model.NewStream()
	for iterator.Next(&stream) {
		stream.Journal.DeleteDate = 0

		if err := service.Save(&stream, "RestoreDeleted stream"); err != nil {
			return derp.Wrap(err, location, "Error restoring deleted stream", stream)
		}

		stream = model.NewStream()
	}

	// No discomfort, no expansion.
	return nil
}

// PurgeDeleted hard deletes all items with the given ancestor that have already been soft-deleted
func (service *Stream) PurgeDeleted(ancestorID primitive.ObjectID) error {

	const location = "service.Stream.PurgeDeleted"

	criteria := exp.Equal("parentIds", ancestorID).AndGreaterThan("journal.deleteDate", 0)

	if err := service.collection.HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Error purging soft-deleted streams")
	}

	return nil
}

// CalcParentIDs scans the parent chain of a stream and generates a "breadcrumbs" slice
// of all of this Stream's parents
func (service *Stream) CalcParentIDs(stream *model.Stream) error {

	// Rule: Notes are always stored under a user's profile, so they have no parents
	if stream.SocialRole == vocab.ObjectTypeNote {
		stream.ParentIDs = id.NewSlice()
		return nil
	}

	// If this stream has no parent, then it has no parent IDs
	if stream.ParentID == primitive.NilObjectID {
		stream.ParentIDs = id.NewSlice()
		return nil
	}

	// Otherwise, load the Parent stream and try to use its parentIDs
	parentStream := model.NewStream()
	if err := service.LoadByID(stream.ParentID, &parentStream); err != nil {
		return derp.Wrap(err, "service.Stream.CalcParentIDs", "Unable to load Parent stream", stream.ParentID)
	}

	// If the parent has no parentIDs, then try to calculate them
	if len(parentStream.ParentIDs) == 0 {
		if err := service.CalcParentIDs(&parentStream); err != nil {
			return derp.Wrap(err, "service.Stream.CalcParentIDs", "Unable to calculate ParentIDs for Parent stream", stream.ParentID)
		}

		// If the parent has been changed, then save that calculation.
		if len(parentStream.ParentIDs) > 0 {
			if err := service.Save(&parentStream, "CalcParentIDs"); err != nil {
				return derp.Wrap(err, "service.Stream.CalcParentIDs", "Unable to save Parent stream", stream.ParentID)
			}
		}
	}

	// Save the ParentIDs to the current stream
	stream.ParentIDs = append(parentStream.ParentIDs, parentStream.StreamID)
	return nil
}

/******************************************
 * WebFinger Behavior
 ******************************************/

func (service *Stream) LoadWebFinger(token string) (digit.Resource, error) {
	return digit.Resource{}, derp.NewBadRequestError("service.Stream.LoadWebFinger", "Not implemented")
}
