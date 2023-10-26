package service

import (
	"context"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/benpate/domain"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Stream manages all interactions with the Stream collection
type Stream struct {
	collection          data.Collection
	templateService     *Template
	userService         *User
	draftService        *StreamDraft
	outboxService       *Outbox
	attachmentService   *Attachment
	host                string
	streamUpdateChannel chan<- model.Stream
}

// NewStream returns a fully populated Stream service.
func NewStream() Stream {
	return Stream{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Stream) Refresh(collection data.Collection, templateService *Template, draftService *StreamDraft, outboxService *Outbox, attachmentService *Attachment, host string, streamUpdateChannel chan model.Stream) {
	service.collection = collection
	service.templateService = templateService
	service.draftService = draftService
	service.outboxService = outboxService
	service.attachmentService = attachmentService

	service.host = host
	service.streamUpdateChannel = streamUpdateChannel
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
	result.URL = service.host + "/" + result.StreamID.Hex()

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
	return service.collection.Iterator(notDeleted(criteria), options...)
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
	stream.URL = service.host + "/" + stream.StreamID.Hex()

	// RULE: Calculate "defaultAllow" groups for this stream.
	defaultTemplate := template.Default()
	defaultRoles := defaultTemplate.AllowedRoles(stream.StateID)
	stream.DefaultAllow = stream.PermissionGroups(defaultRoles...)

	// RULE: Calculate rank
	if stream.Rank == 0 {
		maxRank, err := service.MaxRank(stream.ParentID)

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

// LoadByURL returns a single `Stream` that matches the provided URL
func (service *Stream) LoadByURL(streamURL string, result *model.Stream) error {

	// Verify we have a valid URL
	uri, err := url.Parse(streamURL)

	if err != nil {
		return derp.Wrap(err, "service.Stream.LoadByURL", "Invalid URL", streamURL)
	}

	// Retrieve the Token from the request path
	token, _, err := service.ParsePath(uri)

	if err != nil {
		return derp.Wrap(err, "service.Stream.LoadByURL", "Invalid URL", streamURL)
	}

	return service.LoadByToken(token, result)
}

func (service *Stream) ParsePath(uri *url.URL) (string, string, error) {

	// Verify the URL matches this service
	if domain.AddProtocol(uri.Host) != service.host {
		return "", "", derp.NewBadRequestError("service.Stream.LoadByURL", "Hostname must match this server", uri.String())
	}

	// Load the Stream using the token
	path := list.BySlash(strings.TrimPrefix(uri.Path, "/"))
	token, path := path.Split()

	if token == "" {
		token = "home"
	}

	actionID := path.Head()

	if actionID == "" {
		actionID = "view"
	}

	return token, actionID, nil
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
func (service *Stream) Count(criteria exp.Expression) (int, error) {
	return queries.CountRecords(context.TODO(), service.collection, notDeleted(criteria))
}

// MaxRank returns the maximum rank of all children of a stream
func (service *Stream) MaxRank(parentID primitive.ObjectID) (int, error) {
	return queries.MaxRank(context.TODO(), service.collection, parentID)
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

func (service *Stream) Startup(theme *model.Theme) error {

	// Try to count the number of streams currently in the database
	count, err := service.Count(exp.All())

	if err != nil {
		return derp.Wrap(err, "service.Theme.Startup", "Unable to count streams")
	}

	// If the database is not empty, then do not add more...
	if count > 0 {
		return nil
	}

	streamSchema := service.Schema()

	for _, data := range theme.StartupStreams {

		// Create a new Stream
		stream := model.NewStream()
		if err := streamSchema.SetAll(&stream, data); err != nil {
			derp.Report(derp.Wrap(err, "service.Theme.Startup", "Unable to set stream data", data))
			continue
		}

		// If we have default content, then add that too.
		if content, ok := data["content"].(model.Content); ok {
			stream.Content = content
		}

		// Validate with the general-purpose Stream schema
		if err := streamSchema.Validate(&stream); err != nil {
			derp.Report(derp.Wrap(err, "service.Theme.Startup", "Invalid stream data"))
			continue
		}

		// Get/Validate the template for the new stream
		templateID := data.GetString("templateId")
		template, err := service.templateService.Load(templateID)

		if err != nil {
			derp.Report(derp.Wrap(err, "service.Theme.Startup", "Unable to load template", templateID))
			continue
		}

		// Validate with the specific Template schema
		if err := template.Schema.Validate(&stream); err != nil {
			derp.Report(derp.Wrap(err, "service.Theme.Startup", "Invalid stream data"))
			continue
		}

		// Save the new Stream to the database
		if err := service.Save(&stream, "Created by Startup"); err != nil {
			derp.Report(derp.Wrap(err, "service.Theme.Startup", "Unable to save stream", stream))
			continue
		}
	}

	return nil
}

// Publish marks this stream as "published"
func (service *Stream) Publish(user *model.User, stream *model.Stream) error {

	activityType := vocab.ActivityTypeCreate

	if stream.IsPublished() {
		activityType = vocab.ActivityTypeUpdate
	}

	// RULE: IF this stream is not yet published, then set the publish date
	if stream.PublishDate > time.Now().Unix() {
		stream.PublishDate = time.Now().Unix()
	}

	// RULE: Move unpublish date all the way to the end of time.
	// TODO: LOW: May want to set automatic unpublish dates later...
	stream.UnPublishDate = math.MaxInt64

	// RULE: Set Author to the currently logged in user.
	stream.SetAttributedTo(user.PersonLink())

	// Re-save the Stream with the updated values.
	if err := service.Save(stream, "Publishing"); err != nil {
		return derp.Wrap(err, "service.Stream.Publish", "Error saving stream", stream)
	}

	// Create the Activity to send to the User's Outbox
	activity := mapof.Any{
		"@context": vocab.ContextTypeActivityStreams,
		"id":       stream.ActivityPubURL(),
		"type":     activityType,
		"actor":    user.ActivityPubURL(),
		"object":   stream.GetJSONLD(),
	}

	// Try to publish via the outbox service
	if err := service.outboxService.Publish(user.UserID, stream.URL, activity); err != nil {
		return derp.Wrap(err, "service.Stream.Publish", "Error publishing activity", activity)
	}

	// Done.
	return nil
}

// UnPublish marks this stream as "published"
func (service *Stream) UnPublish(user *model.User, stream *model.Stream) error {

	// RULE: Move unpublish date all the way to the end of time.
	stream.UnPublishDate = time.Now().Unix()

	// Re-save the Stream with the updated values.
	if err := service.Save(stream, "Publish"); err != nil {
		return derp.Wrap(err, "service.Stream.UnPublish", "Error saving stream", stream)
	}

	// Create the Activity to send to the User's Outbox
	activity := mapof.Any{
		"@context": vocab.ContextTypeActivityStreams,
		"type":     vocab.ActivityTypeDelete,
		"actor":    user.ActivityPubURL(),
		"object":   stream.GetJSONLD(),
	}

	// Remove the record from the inbox
	if err := service.outboxService.UnPublish(user.UserID, stream.URL, activity); err != nil {
		return derp.Wrap(err, "service.Stream.UnPublish", "Error removing from outbox", stream)
	}

	// Done.
	return nil
}

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
	criteria := exp.Equal("parentIds", ancestorID).AndGreaterThan("deleteDate", 0)
	iterator, err := service.collection.Iterator(criteria)

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

	criteria := exp.Equal("parentIds", ancestorID).AndGreaterThan("deleteDate", 0)

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

// UserCan checks a user's permission to perform an action on a Stream.  If not allowed,
// then the returned error describes why the access was denied.
func (service *Stream) UserCan(authorization *model.Authorization, stream *model.Stream, actionID string) error {

	const location = "service.Stream.UserCan"

	// Find the Template used by this stream
	template, err := service.templateService.Load(stream.TemplateID)

	if err != nil {
		return derp.Wrap(err, location, "Invalid Template")
	}

	// Find the action that the user wants to perform
	action, ok := template.Action(actionID)

	if !ok {
		return derp.NewBadRequestError(location, "Invalid Action", actionID)
	}

	// Check permissions on the action
	if !action.UserCan(stream, authorization) {
		return derp.NewUnauthorizedError(location, "User is not authorized to perform this action", actionID)
	}

	// UserCan!
	return nil
}

/******************************************
 * WebFinger Behavior
 ******************************************/

func (service *Stream) LoadWebFinger(token string) (digit.Resource, error) {
	return digit.Resource{}, derp.NewBadRequestError("service.Stream.LoadWebFinger", "Not implemented")
}

/******************************************
 * Mastodon API
 ******************************************/

func (service *Stream) QueryByUser(userID primitive.ObjectID, criteria exp.Expression, options ...option.Option) ([]model.Stream, error) {

	criteria = criteria.AndEqual("ownerId", userID)
	options = append(options, option.SortDesc("createDate"))

	return service.Query(criteria, options...)
}
