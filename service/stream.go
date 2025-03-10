package service

import (
	"context"
	"iter"
	"net/url"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/EmissarySocial/emissary/tools/parse"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/domain"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/mediaserver"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/html"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Stream manages all interactions with the Stream collection
type Stream struct {
	collection        data.Collection
	domainService     *Domain
	searchTagService  *SearchTag
	templateService   *Template
	draftService      *StreamDraft
	outboxService     *Outbox
	attachmentService *Attachment
	activityStream    *ActivityStream
	contentService    *Content
	keyService        *EncryptionKey
	followerService   *Follower
	ruleService       *Rule
	userService       *User
	webhookService    *Webhook
	host              string
	mediaserver       mediaserver.MediaServer
	queue             *queue.Queue
	sseUpdateChannel  chan<- primitive.ObjectID
}

// NewStream returns a fully populated Stream service.
func NewStream() Stream {
	return Stream{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Stream) Refresh(collection data.Collection, domainService *Domain, searchTagService *SearchTag, templateService *Template, draftService *StreamDraft, outboxService *Outbox, attachmentService *Attachment, activityStream *ActivityStream, contentService *Content, keyService *EncryptionKey, followerService *Follower, ruleService *Rule, userService *User, webhookService *Webhook, mediaserver mediaserver.MediaServer, queue *queue.Queue, host string, sseUpdateChannel chan primitive.ObjectID) {
	service.collection = collection
	service.domainService = domainService
	service.searchTagService = searchTagService
	service.templateService = templateService
	service.draftService = draftService
	service.outboxService = outboxService
	service.attachmentService = attachmentService
	service.activityStream = activityStream
	service.contentService = contentService
	service.keyService = keyService
	service.followerService = followerService
	service.ruleService = ruleService
	service.userService = userService
	service.webhookService = webhookService
	service.mediaserver = mediaserver
	service.queue = queue

	service.host = host
	service.sseUpdateChannel = sseUpdateChannel
}

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

// Close stops any background processes controlled by this service
func (service *Stream) Close() {

}

/******************************************
 * Common Methods
 ******************************************/

// New returns a new Stream that uses the named template.
func (service *Stream) New() model.Stream {
	result := model.NewStream()
	result.URL = service.host + "/" + result.Token
	// TODO: HIGH: Use stream Template schema to set default values in the new stream.

	return result
}

// Count returns the number of records that match the provided criteria
func (service *Stream) Count(criteria exp.Expression) (int64, error) {
	return service.collection.Count(notDeleted(criteria))
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

// Range returns a Go 1.23 RangeFunc that iterates over the Streams that match the provided criteria
func (service *Stream) Range(criteria exp.Expression, options ...option.Option) (iter.Seq[model.Stream], error) {

	iter, err := service.List(criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Stream.Range", "Error creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewStream), nil
}

// RangeSummary returns a Go 1.23 RangeFunc that iterates over the Stream Summaries that match the provided criteria
func (service *Stream) RangeSummary(criteria exp.Expression, options ...option.Option) (iter.Seq[model.StreamSummary], error) {

	iter, err := service.List(criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Stream.Range", "Error creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewStreamSummary), nil
}

// List returns an iterator containing all of the Streams that match the provided criteria
func (service *Stream) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Load retrieves an Stream from the database
func (service *Stream) Load(criteria exp.Expression, stream *model.Stream) error {

	if err := service.collection.Load(notDeleted(criteria), stream); err != nil {
		return derp.Wrap(err, "service.Stream.Load", "Error loading Stream", criteria)
	}

	return nil
}

// Save adds/updates an Stream in the database
func (service *Stream) Save(stream *model.Stream, note string) error {

	const location = "service.Stream.Save"

	template, err := service.templateService.Load(stream.TemplateID)

	if err != nil {
		return derp.Wrap(err, location, "Invalid Template", stream.TemplateID)
	}

	// Track changes to key status fields
	wasNew := stream.IsNew()

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

	// Validate the value (using the global stream schema) before saving
	if err := service.Schema().Validate(stream); err != nil {
		return derp.Wrap(err, "service.Stream.Save", "Error validating Stream using StreamSchema", stream)
	}

	// Validate the value (using the template-specific schema) before saving
	if err := template.Schema.Validate(stream); err != nil {
		return derp.Wrap(err, "service.Stream.Save", "Error validating Stream using TemplateSchema", stream)
	}

	// RULE: If this stream is not a profile stream and does not have ParentIDs, then calculate them now.
	if (stream.NavigationID != "profile") && (len(stream.ParentIDs) == 0) {
		if err := service.CalcParentIDs(stream); err != nil {
			return derp.Wrap(err, location, "Error calculating parent IDs", stream)
		}
	}

	// RULE: Calculate the stream context
	service.CalcContext(stream)

	// Try to save the Stream to the database
	if err := service.collection.Save(stream, note); err != nil {
		return derp.Wrap(err, location, "Error saving Stream", stream, note)
	}

	// Send stream:create and stream:update Webhooks
	if wasNew {
		service.webhookService.Send(stream, model.WebhookEventStreamCreate)
	} else {
		service.webhookService.Send(stream, model.WebhookEventStreamUpdate)
	}

	if stream.IsPublished() && stream.Syndication.IsChanged() {
		if err := service.sendSyndicationMessages(stream, stream.Syndication.Added, stream.Syndication.Deleted); err != nil {
			return derp.Wrap(err, location, "Error sending syndication messages", stream)
		}
	}

	// NON-BLOCKING: Notify other processes on this server that the stream has been updated
	go func() {
		service.sseUpdateChannel <- stream.StreamID
		service.sseUpdateChannel <- stream.ParentID
	}()

	// One milisecond delay prevents overlapping stream.CreateDates.  Deal with it.
	// TODO: There has to be a better way than this...
	time.Sleep(1 * time.Millisecond)

	return nil
}

// Delete removes an Stream from the database (virtual delete)
func (service *Stream) Delete(stream *model.Stream, note string) error {

	const location = "service.Stream.Delete"

	// Delete this Stream
	if err := service.collection.Delete(stream, note); err != nil {
		return derp.Wrap(err, location, "Error deleting Stream", stream, note)
	}

	// Delete related records -- this can happen in the background
	go func() {

		// Send Webhooks (if configured)
		service.webhookService.Send(stream, model.WebhookEventStreamDelete)

		if stream.IsPublished() {
			service.webhookService.Send(stream, model.WebhookEventStreamPublishUndo)

			if err := service.sendSyndicationMessages(stream, nil, stream.Syndication.Values); err != nil {
				derp.Report(derp.Wrap(err, location, "Error sending syndication messages", stream))
			}
		}

		// RULE: Delete all related Children
		if err := service.DeleteByParent(stream.StreamID, note); err != nil {
			derp.Report(derp.Wrap(err, location, "Error deleting child streams", stream, note))
		}

		// RULE: Delete all related Attachments
		if err := service.attachmentService.DeleteAll(model.AttachmentObjectTypeStream, stream.StreamID, note); err != nil {
			derp.Report(derp.Wrap(err, location, "Error deleting attachments", stream, note))
		}

		// RULE: Delete all related Drafts
		if err := service.draftService.Delete(stream, note); err != nil {
			derp.Report(derp.Wrap(err, location, "Error deleting drafts", stream, note))
		}

		// RULE: Delete Outbox Messages
		if err := service.outboxService.DeleteByParentID(model.FollowerTypeStream, stream.StreamID); err != nil {
			derp.Report(derp.Wrap(err, location, "Error deleting outbox messages", stream, note))
		}

		// NON-BLOCKING: Notify other processes on this server that the stream has been updated
		service.sseUpdateChannel <- stream.ParentID

	}()

	// Bueno!!
	return nil
}

// DeleteMany removes all child streams from the provided stream (virtual delete)
func (service *Stream) DeleteMany(criteria exp.Expression, note string) error {

	const location = "service.Stream.DeleteMany"

	it, err := service.List(notDeleted(criteria))

	if err != nil {
		return derp.Wrap(err, location, "Error listing streams to delete", criteria)
	}

	stream := model.NewStream()

	for it.Next(&stream) {
		if err := service.Delete(&stream, note); err != nil {
			return derp.Wrap(err, location, "Error deleting stream", stream)
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
	return schema.New(model.StreamSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

// RangeAll returns a RangeFunc over all streams
func (service *Stream) RangeAll() (iter.Seq[model.StreamSummary], error) {
	return service.RangeSummary(exp.All())
}

// RangePublished returns a RangeFunc over all streams that are currently published
func (service *Stream) RangePublished() (iter.Seq[model.Stream], error) {

	now := time.Now().Unix()

	criteria := exp.LessOrEqual("publishDate", now).
		AndGreaterOrEqual("unpublishDate", now)

	return service.Range(criteria)
}

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

// ListPublishedByParent returns all Streams that match a particular parentID
func (service *Stream) ListPublishedByParent(parentID primitive.ObjectID) (data.Iterator, error) {

	now := time.Now().Unix()

	criteria := exp.LessOrEqual("publishDate", now).
		AndGreaterOrEqual("unpublishDate", now).
		AndEqual("parentId", parentID)

	return service.List(criteria, option.SortDesc("publishDate"))
}

// ListByTemplate returns all `Streams` that use a particular `Template`
func (service *Stream) ListByTemplate(template string) (data.Iterator, error) {
	return service.List(exp.Equal("templateId", template))
}

// QueryByParentAndDate returns a slice of Streams that are DIRECT CHILDREN of the provided StreamID
func (service *Stream) QueryByParentAndDate(streamID primitive.ObjectID, publishedDate int64, pageSize int) ([]model.Stream, error) {
	criteria := exp.Equal("parentId", streamID).AndLessThan("publishDate", publishedDate)
	return service.Query(criteria, option.SortDesc("publishDate"), option.MaxRows(int64(pageSize)))
}

// QueryByParentAndDate returns a slice of Streams that are ANY DEPTH below the provided StreamID
func (service *Stream) QueryByAncestorAndDate(streamID primitive.ObjectID, publishedDate int64, pageSize int) ([]model.Stream, error) {
	criteria := exp.Equal("parentIds", streamID).AndLessThan("publishDate", publishedDate)
	return service.Query(criteria, option.SortDesc("publishDate"), option.MaxRows(int64(pageSize)))
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
	return service.attachmentService.LoadFirstByObjectID(model.AttachmentObjectTypeStream, streamID)
}

// MaxRank returns the maximum rank of all children of a stream
func (service *Stream) MaxRank(parentID primitive.ObjectID) (int, error) {
	return queries.MaxRank(context.TODO(), service.collection, parentID)
}

/******************************************
 * Initialization Actions
 ******************************************/

// SetLocationTop sets a Stream to be a top-level navigation item
func (service *Stream) SetLocationTop(template *model.Template, stream *model.Stream) error {

	// RULE: Template must be allowed in the Top
	if !template.CanBeContainedBy("top") {
		return derp.NewBadRequestError("service.Stream.SetLocationTop", "Template cannot be contained by 'top'", template)
	}

	// Set values in the Stream
	stream.TemplateID = template.TemplateID
	stream.NavigationID = stream.StreamID.Hex()
	stream.ParentID = primitive.NilObjectID
	stream.ParentIDs = make([]primitive.ObjectID, 0)
	stream.ParentTemplateID = ""
	return nil
}

// SetLocationInbox sets a Stream's location to be a User's outbox
func (service *Stream) SetLocationOutbox(template *model.Template, stream *model.Stream, userID primitive.ObjectID) error {

	const location = "service.Stream.SetLocationOutbox"

	// RULE: Valid User is Required
	if userID.IsZero() {
		return derp.NewUnauthorizedError(location, "User ID is required")
	}

	// RULE: Template must be allowed in the Outbox
	if !template.CanBeContainedBy("outbox") {
		return derp.NewBadRequestError(location, "Template cannot be contained by 'outbox'", template)
	}

	// Set values in the Stream
	stream.TemplateID = template.TemplateID
	stream.NavigationID = "profile"
	stream.ParentID = userID
	stream.ParentIDs = []primitive.ObjectID{}
	stream.ParentTemplateID = ""

	return nil
}

// SetLocationChild sets a Stream to be a child of another Stream
func (service *Stream) SetLocationChild(template *model.Template, stream *model.Stream, parent *model.Stream) error {

	const location = "service.Stream.SetLocationChild"

	// Get the Parent Template
	parentTemplate, err := service.templateService.Load(parent.TemplateID)

	if err != nil {
		return derp.Wrap(err, location, "Invalid Parent Template", parent)
	}

	// RULE: Template must be allowed in the Parent
	if !template.CanBeContainedBy(parentTemplate.TemplateRole) {
		return derp.NewBadRequestError(location, "Template cannot be contained by parent", template, parent)
	}

	// Set values in the Stream
	stream.TemplateID = template.TemplateID
	stream.NavigationID = parent.NavigationID
	stream.ParentID = parent.StreamID
	stream.ParentIDs = append(parent.ParentIDs, parent.StreamID)
	stream.ParentTemplateID = parent.TemplateID

	return nil
}

/******************************************
 * Custom Actions
 ******************************************/

func (service *Stream) SetAttributedTo(user *model.User) {
	err := queries.SetAttributedTo(context.Background(), service.collection, user.PersonLink())
	derp.Report(derp.Wrap(err, "service.Stream.SetAttributedTo", "Error setting attributedTo"))
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

// ParsePathextracts the Stream token and actionID from a URL
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

// ParseURL validates that a URL matches the current server, and then extracts the streamID from it.
func (service *Stream) ParseURL(streamURL string) (primitive.ObjectID, error) {

	const location = "service.Stream.ParseURL"

	parsedURL, err := url.Parse(streamURL)

	if err != nil {
		return primitive.NilObjectID, derp.Wrap(err, location, "Invalid URL", streamURL)
	}

	// Get the first part of the path (which is the stream ID or token)
	path := strings.TrimPrefix(parsedURL.Path, "/")
	path, _, _ = strings.Cut(path, "/")

	// If the value looks like an ObjectID, then return it
	if streamID, err := primitive.ObjectIDFromHex(path); err == nil {
		return streamID, nil
	}

	// Otherwise, try to load the stream by Token
	stream := model.NewStream()
	if err := service.LoadByToken(path, &stream); err != nil {
		return primitive.NilObjectID, derp.Wrap(err, location, "Invalid Token", path)
	}

	return stream.StreamID, nil
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

// CalcContext calculates the conversational context for a given stream,
// IF it can be determined.
func (service *Stream) CalcContext(stream *model.Stream) {

	// If this is an original stream (not a reply) then its context is itself.
	if stream.InReplyTo == "" {
		stream.Context = stream.ActivityPubURL()
		return
	}

	// Load the "InReplyTo" document from the ActivityStream and use its
	// context.  Note: this should have been calculated already via the
	// ascontextmaker client.
	document, _ := service.activityStream.Load(stream.InReplyTo)

	if context := document.Context(); context != "" {
		stream.Context = document.Context()
		return
	}

	// If a context could not be assigned, then use the InReplyTo value instead.
	stream.Context = stream.InReplyTo
}

func (service *Stream) CalculateTags(stream *model.Stream) {

	const location = "service.Stream.CalculateTags"

	// Load the template (to get the tag paths)
	template, err := service.templateService.Load(stream.TemplateID)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Unable to load Template", stream.TemplateID))
		return
	}

	// Get values for each tag path in the Stream
	schema := service.Schema()
	hashtags := sliceof.NewString()

	for _, path := range template.TagPaths {

		if value, err := schema.Get(stream, path); err == nil {

			// Massage the value into a cleanly searchable string
			stringValue := convert.String(value)
			stringValue = html.ToSearchText(stringValue)
			hashtags = append(hashtags, parse.Hashtags(stringValue)...)
		}
	}

	// Normalize Hashtag names by looking them up in the database
	hashtagNames, _, err := service.searchTagService.NormalizeTags(hashtags...)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Error normalizing tags"))
	}

	// Apply the #hashtags back to the Stream
	stream.Hashtags = hashtagNames
}

/******************************************
 * SearchResulter Interface
 ******************************************/

// SearchResult returns a SearchResult object that represents this Stream in the search index
func (service *Stream) SearchResult(stream *model.Stream) model.SearchResult {

	result := model.NewSearchResult()

	// If the stream has been published, then try to generate a SearchResult for it.
	if stream.IsPublished() {

		// Try to generate the searchResult.FullText using the Template for this Stream
		if template, err := service.templateService.Load(stream.TemplateID); err == nil {

			// If SearchOptions are not empty, then Streams using this Template are searchable
			if len(template.SearchOptions) > 0 {

				result.URL = stream.URL
				result.Tags = slice.Map(stream.Hashtags, model.ToToken)
				result.Type = firstOf(template.SearchOptions.Execute("type", stream), template.SocialRole)
				result.Name = firstOf(template.SearchOptions.Execute("name", stream), stream.Label)
				result.AttributedTo = firstOf(template.SearchOptions.Execute("attributedTo", stream), stream.AttributedTo.Name)
				result.Summary = firstOf(template.SearchOptions.Execute("summary", stream), stream.Summary)
				result.IconURL = firstOf(template.SearchOptions.Execute("iconUrl", stream), stream.IconURL)
				result.Text = template.SearchOptions.Execute("text", stream)
				result.Date = stream.StartDate.Time

				if place := stream.Places.First(); place.NotEmpty() {
					result.Place = place.GeoJSON()
				}

				if tagString := template.SearchOptions.Execute("tags", stream); tagString != "" {
					tags := strings.Split(tagString, " ")
					result.Tags = append(result.Tags, tags...)
				}

				return result
			}
		}
	}

	// Fall through means this Stream is not searchable
	result.URL = stream.URL
	result.DeleteDate = time.Now().Unix()
	return result
}
