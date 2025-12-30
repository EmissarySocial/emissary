package service

import (
	"iter"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/EmissarySocial/emissary/realtime"
	"github.com/EmissarySocial/emissary/tools/datetime"
	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/EmissarySocial/emissary/tools/parse"
	"github.com/EmissarySocial/emissary/tools/random"
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	dt "github.com/benpate/domain"
	"github.com/benpate/exp"
	"github.com/benpate/geo"
	"github.com/benpate/mediaserver"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/delta"
	"github.com/benpate/rosetta/html"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Stream manages all interactions with the Stream collection
type Stream struct {
	factory           *Factory
	attachmentService *Attachment
	circleService     *Circle
	contentService    *Content
	domainService     *Domain
	draftService      *StreamDraft
	geocodeService    GeocodeAddress
	importService     *Import
	importItemService *ImportItem
	keyService        *EncryptionKey
	mentionService    *Mention
	outboxService     *Outbox
	searchTagService  *SearchTag
	templateService   *Template
	followerService   *Follower
	ruleService       *Rule
	userService       *User
	webhookService    *Webhook
	host              string
	mediaserver       mediaserver.MediaServer
	queue             *queue.Queue
	sseUpdateChannel  chan<- realtime.Message
}

// NewStream returns a fully populated Stream service.
func NewStream(factory *Factory) Stream {
	return Stream{
		factory: factory,
	}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Stream) Refresh(attachmentService *Attachment, circleService *Circle, contentService *Content, domainService *Domain, draftService *StreamDraft, followerService *Follower, geocodeService GeocodeAddress, importService *Import, importItemService *ImportItem, keyService *EncryptionKey, mentionService *Mention, outboxService *Outbox, ruleService *Rule, searchTagService *SearchTag, templateService *Template, userService *User, webhookService *Webhook, mediaserver mediaserver.MediaServer, queue *queue.Queue, sseUpdateChannel chan<- realtime.Message, host string) {
	service.attachmentService = attachmentService
	service.circleService = circleService
	service.contentService = contentService
	service.domainService = domainService
	service.draftService = draftService
	service.followerService = followerService
	service.geocodeService = geocodeService
	service.importService = importService
	service.importItemService = importItemService
	service.keyService = keyService
	service.mentionService = mentionService
	service.outboxService = outboxService
	service.ruleService = ruleService
	service.searchTagService = searchTagService
	service.templateService = templateService
	service.userService = userService
	service.webhookService = webhookService
	service.mediaserver = mediaserver
	service.queue = queue

	service.host = host
	service.sseUpdateChannel = sseUpdateChannel
}

func (service *Stream) Startup(session data.Session, theme *model.Theme) error {

	// Try to count the number of streams currently in the database
	count, err := service.Count(session, exp.All())

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

		// Set this Stream as "Published"
		stream.PublishDate = 0

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
		if err := service.Save(session, &stream, "Created by Startup"); err != nil {
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

func (service *Stream) collection(session data.Session) data.Collection {
	return session.Collection("Stream")
}

// New returns a new Stream that uses the named template.
func (service *Stream) New() model.Stream {
	result := model.NewStream()
	result.URL = service.host + "/" + result.Token
	// TODO: HIGH: Use stream Template schema to set default values in the new stream.

	return result
}

// Count returns the number of records that match the provided criteria
func (service *Stream) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns an slice containing all of the Streams that match the provided criteria
func (service *Stream) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.Stream, error) {
	result := make([]model.Stream, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// QuerySummary returns an slice containing StreamSummaries for all of the Streams that match the provided criteria
func (service *Stream) QuerySummary(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.StreamSummary, error) {
	result := make([]model.StreamSummary, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

func (service *Stream) QueryIDOnly(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.IDOnly, error) {
	result := make([]model.IDOnly, 0)
	options = append(options, option.Fields("_id"))
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// Range returns a Go 1.23 RangeFunc that iterates over the Streams that match the provided criteria
func (service *Stream) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.Stream], error) {

	iter, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Stream.Range", "Unable to create iterator", criteria)
	}

	return RangeFunc(iter, model.NewStream), nil
}

// RangeSummary returns a Go 1.23 RangeFunc that iterates over the Stream Summaries that match the provided criteria
func (service *Stream) RangeSummary(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.StreamSummary], error) {

	const location = "service.Stream.RangeSummary"

	// NILCHECK: Service cannot be nil
	if service == nil {
		return nil, derp.Internal(location, "Service cannot be nil. This should never happen.")
	}

	// NILCHECK: Session cannot be nil
	if session == nil {
		return nil, derp.BadRequest(location, "Session cannot be nil. This should never happen.")
	}

	// Get an iterator from the database
	iter, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to create iterator", criteria)
	}

	// Convert it into a RangeFunc
	return RangeFunc(iter, model.NewStreamSummary), nil
}

// List returns an iterator containing all of the Streams that match the provided criteria
func (service *Stream) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {

	const location = "service.Stream.List"

	// NILCHECK: Service cannot be nil
	if service == nil {
		return nil, derp.Internal(location, "Service cannot be nil. This should never happen.")
	}

	// NILCHECK: Session cannot be nil
	if session == nil {
		return nil, derp.BadRequest(location, "Session cannot be nil. This should never happen.")
	}

	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Load retrieves an Stream from the database
func (service *Stream) Load(session data.Session, criteria exp.Expression, stream *model.Stream) error {

	const location = "service.Stream.Load"

	// NILCHECK: Service cannot be nil
	if service == nil {
		return derp.Internal(location, "Service cannot be nil. This should never happen.")
	}

	// NILCHECK: Stream cannot be nil
	if session == nil {
		return derp.BadRequest(location, "Session cannot be nil. This should never happen.")
	}

	// Load the Stream from the database
	if err := service.collection(session).Load(notDeleted(criteria), stream); err != nil {
		return derp.Wrap(err, location, "Unable to load Stream", criteria)
	}

	return nil
}

// Save adds/updates an Stream in the database
func (service *Stream) Save(session data.Session, stream *model.Stream, note string) error {

	const location = "service.Stream.Save"

	// NILCHECK: Service cannot be nil
	if service == nil {
		return derp.Internal(location, "Service cannot be nil. This should never happen.")
	}

	// NILCHECK: Stream cannot be nil
	if session == nil {
		return derp.BadRequest(location, "Session cannot be nil. This should never happen.")
	}

	// NILCHECK: Stream cannot be nil
	if stream == nil {
		return derp.BadRequest(location, "Stream cannot be nil. This should never happen.")
	}

	// Track changes to key status fields
	wasNew := stream.IsNew()

	// RULE: Calculate rank
	if stream.Rank == 0 {
		maxRank, err := service.MaxRank(session, stream.ParentID)

		if err != nil {
			return derp.Wrap(err, location, "Unable to calculate max rank")
		}
		stream.Rank = maxRank
	}

	// RULE: If unassigned, shuffle the stream after the first trillion other results (will reset in 1 hour)
	if stream.Shuffle == 0 {
		stream.Shuffle = math.MaxInt64 - int64(random.GenerateInt(1, 999_999_999_999))
	}

	// RULE: Default Token
	if stream.Token == "" {
		stream.Token = stream.StreamID.Hex()
	}

	// Geocode the Location (if needed)
	if stream.Location.NotZero() {
		if err := service.geocodeService.GeocodeAndQueue(session, stream); err != nil {
			return derp.Wrap(err, location, "Unable to geocode stream", stream.Location)
		}
	}

	// If this stream has anything but a NIL templateID
	if stream.TemplateID != "" {

		// Load the template used by this Stream
		template, err := service.templateService.Load(stream.TemplateID)

		if err != nil {
			return derp.Wrap(err, location, "Unable to load template", stream.TemplateID)
		}

		// Copy default values from the Template
		stream.SocialRole = template.SocialRole
		stream.IsSubscribable = template.IsSubscribable()
		stream.URL = service.host + "/" + stream.StreamID.Hex()

		// RULE: Calculate "defaultAllow" groups for this stream.
		service.calcDefaultAllow(&template, stream)

		// Validate the value (using the template-specific schema) before saving
		if err := template.Schema.Validate(stream); err != nil {
			return derp.Wrap(err, location, "Invalid Stream: using TemplateSchema", stream)
		}
	}

	// Validate the value (using the global stream schema) before saving
	if err := service.Schema().Validate(stream); err != nil {
		return derp.Wrap(err, location, "Invalid Stream: using StreamSchema", stream)
	}

	// RULE: calculate Parent IDs
	service.calcParentIDs(session, stream)

	// RULE: Calculate the stream context
	service.calcContext(stream)

	// RULE: Calculate privileges for this stream
	service.calcPrivilegeIDs(stream)

	// Try to save the Stream to the database
	if err := service.collection(session).Save(stream, note); err != nil {
		return derp.Wrap(err, location, "Unable to save Stream", stream, note)
	}

	// Send SSE notifications to `InReplyTo` streams (if possible)
	service.NotifyInReplyTo(session, stream.InReplyTo)

	// Send stream:create and stream:update Webhooks
	eventName := iif(wasNew, model.WebhookEventStreamCreate, model.WebhookEventStreamUpdate)
	service.webhookService.Send(stream, eventName)

	return nil
}

// HardDeleteByID removes a specific Stream record, without applying any additional business rules
func (service *Stream) HardDeleteByID(session data.Session, streamID primitive.ObjectID) error {

	const location = "service.Stream.HardDeleteByID"

	criteria := exp.Equal("_id", streamID)

	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Unable to delete Stream")
	}

	return nil
}

// Delete removes an Stream from the database (virtual delete)
func (service *Stream) Delete(session data.Session, stream *model.Stream, note string) error {

	const location = "service.Stream.Delete"

	// Delete this Stream
	if err := service.collection(session).Delete(stream, note); err != nil {
		return derp.Wrap(err, location, "Unable to delete Stream from database", stream, note)
	}

	// Send Webhooks (if configured)
	service.webhookService.Send(stream, model.WebhookEventStreamDelete)

	if stream.IsPublished() {
		service.webhookService.Send(stream, model.WebhookEventStreamPublishUndo)

		if err := service.sendSyndicationMessages(stream, nil, nil, stream.Syndication.Values); err != nil {
			derp.Report(derp.Wrap(err, location, "Unable to send syndication messages", stream))
		}
	}

	// RULE: Delete all related Children
	if err := service.DeleteByParent(session, stream.StreamID, note); err != nil {
		derp.Report(derp.Wrap(err, location, "Unable to delete child streams", stream, note))
	}

	// RULE: Delete all related Attachments
	if err := service.attachmentService.DeleteAll(session, model.AttachmentObjectTypeStream, stream.StreamID, note); err != nil {
		derp.Report(derp.Wrap(err, location, "Unable to delete attachments", stream, note))
	}

	// RULE: Delete all related Drafts
	if err := service.draftService.Delete(session, stream, note); err != nil {
		derp.Report(derp.Wrap(err, location, "Unable to delete drafts", stream, note))
	}

	// RULE: Delete Outbox Messages
	if err := service.outboxService.DeleteByParentID(session, model.FollowerTypeStream, stream.StreamID); err != nil {
		derp.Report(derp.Wrap(err, location, "Unable to delete outbox messages", stream, note))
	}

	// NON-BLOCKING: Notify other processes on this server that the stream has been updated
	go func() {
		service.sseUpdateChannel <- realtime.NewMessage_ChildUpdated(stream.ParentID)
	}()

	// Bueno!!
	return nil
}

// DeleteMany removes all child streams from the provided stream (virtual delete)
func (service *Stream) DeleteMany(session data.Session, criteria exp.Expression, note string) error {

	const location = "service.Stream.DeleteMany"

	it, err := service.List(session, criteria)

	if err != nil {
		return derp.Wrap(err, location, "Unable to list streams to delete", criteria)
	}

	for stream := model.NewStream(); it.Next(&stream); stream = model.NewStream() {
		if err := service.Delete(session, &stream, note); err != nil {
			return derp.Wrap(err, location, "Unable to delete stream", stream)
		}
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

func (service *Stream) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

func (service *Stream) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewStream()
	err := service.Load(session, criteria, &result)
	return &result, err
}

func (service *Stream) ObjectSave(session data.Session, object data.Object, note string) error {

	if stream, ok := object.(*model.Stream); ok {
		return service.Save(session, stream, note)
	}
	return derp.Internal("service.Stream.ObjectSave", "Invalid object type", object)
}

func (service *Stream) ObjectDelete(session data.Session, object data.Object, note string) error {
	if stream, ok := object.(*model.Stream); ok {
		return service.Delete(session, stream, note)
	}
	return derp.Internal("service.Stream.ObjectDelete", "Invalid object type", object)
}

func (service *Stream) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.Unauthorized("service.Stream", "Not Authorized")
}

func (service *Stream) Schema() schema.Schema {
	return schema.New(model.StreamSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

// RangePublished returns a RangeFunc over all streams that are currently published
func (service *Stream) RangePublished(session data.Session) (iter.Seq[model.Stream], error) {

	now := time.Now().Unix()

	criteria := exp.LessOrEqual("publishDate", now).
		AndGreaterOrEqual("unpublishDate", now)

	return service.Range(session, criteria)
}

// ListNavigation returns all Streams of type FOLDER at the top of the hierarchy
func (service *Stream) ListNavigation(session data.Session) (data.Iterator, error) {
	return service.List(
		session,
		exp.Equal("parentId", primitive.NilObjectID),
		option.SortAsc("rank"),
	)
}

// RangeByParent returns an iterator that contains all child streams of the provided parentID
func (service *Stream) RangeByParent(session data.Session, parentID primitive.ObjectID) (iter.Seq[model.Stream], error) {
	return service.Range(session, exp.Equal("parentId", parentID))
}

// RangeByParentIDs returns an iterator that contains a descendant (at any level) of the provided parentID
func (service *Stream) RangeByParentIDs(session data.Session, parentID primitive.ObjectID) (iter.Seq[model.Stream], error) {
	return service.Range(session, exp.Equal("parentIds", parentID))
}

func (service *Stream) RangeByPrivileges(session data.Session, privileges ...primitive.ObjectID) (iter.Seq[model.Stream], error) {

	const location = "service.Stream.RangeByPrivilege"

	// RULE: PrivilegeID is required
	if len(privileges) == 0 {
		return nil, derp.BadRequest(location, "Query must have at least one Privilege")
	}

	criteria := exp.In("privilegeIds", privileges)

	return service.Range(session, criteria)
}

// ListPublishedByParent returns all Streams that match a particular parentID
func (service *Stream) ListPublishedByParent(session data.Session, parentID primitive.ObjectID) (data.Iterator, error) {

	const location = "service.Stream.ListPublishedByParent"

	// RULE: ParentID is required
	if parentID.IsZero() {
		return nil, derp.BadRequest(location, "ParentID is required")
	}

	now := time.Now().Unix()

	criteria := exp.LessOrEqual("publishDate", now).
		AndGreaterOrEqual("unpublishDate", now).
		AndEqual("parentId", parentID)

	return service.List(session, criteria, option.SortDesc("publishDate"))
}

// ListByTemplate returns all `Streams` that use a particular `Template`
func (service *Stream) ListByTemplate(session data.Session, template string) (data.Iterator, error) {

	const location = "service.Stream.ListByTemplate"

	// RULE: Template is required
	if template == "" {
		return nil, derp.BadRequest(location, "Template is required")
	}

	return service.List(session, exp.Equal("templateId", template))
}

// QuerySubscribable returns all Streams in a User's outbox that are subscribe-able
func (service *Stream) QuerySubscribable(session data.Session, userID primitive.ObjectID) ([]model.StreamSummary, error) {

	const location = "service.Stream.QuerySubscribable"

	// RULE: UserID is required
	if userID.IsZero() {
		return nil, derp.BadRequest(location, "UserID is required")
	}

	criteria := exp.Equal("parentId", userID).AndEqual("isSubscribable", true)
	return service.QuerySummary(session, criteria, option.SortAsc("templateId"), option.SortAsc("label"))
}

// QueryByParentAndDate returns a slice of Streams that are DIRECT CHILDREN of the provided StreamID
func (service *Stream) QueryByParentAndDate(session data.Session, parentID primitive.ObjectID, publishedDate int64, pageSize int) ([]model.Stream, error) {

	const location = "service.Stream.QueryByParentAndDate"

	// RULE: ParentID is required
	if parentID.IsZero() {
		return nil, derp.BadRequest(location, "ParentID is required")
	}

	criteria := exp.Equal("parentId", parentID).AndLessThan("publishDate", publishedDate)
	return service.Query(session, criteria, option.SortDesc("publishDate"), option.MaxRows(int64(pageSize)))
}

// QueryByParentAndDate returns a slice of Streams that are ANY DEPTH below the provided StreamID
func (service *Stream) QueryByAncestorAndDate(session data.Session, streamID primitive.ObjectID, publishedDate int64, pageSize int) ([]model.Stream, error) {

	const location = "service.Stream.QueryByAncestorAndDate"

	// RULE: StreamID is required
	if streamID.IsZero() {
		return nil, derp.BadRequest(location, "StreamID is required")
	}

	criteria := exp.Equal("parentIds", streamID).AndLessThan("publishDate", publishedDate)
	return service.Query(session, criteria, option.SortDesc("publishDate"), option.MaxRows(int64(pageSize)))
}

// QueryFeaturedByUser returns all Streams in a particular User's outbox that have been featured.
func (service *Stream) QueryFeaturedByUser(session data.Session, userID primitive.ObjectID) ([]model.StreamSummary, error) {

	const location = "service.Stream.QueryFeaturedByUser"

	// RULE: UserID is required
	if userID.IsZero() {
		return nil, derp.BadRequest(location, "UserID is required")
	}

	criteria := exp.Equal("parentId", userID).AndEqual("isFeatured", true)

	return service.QuerySummary(
		session,
		criteria,
		option.SortDesc("publishDate"),
		option.Fields("url"),
	)
}

// QueryByPrivilege returns all Streams that are associated with a particular PrivilegeID
func (service *Stream) QueryByPrivilege(session data.Session, privilegeIDs ...primitive.ObjectID) ([]model.Stream, error) {

	const location = "service.Stream.QueryByPrivilege"

	// RULE: PrivilegeID is required
	if len(privilegeIDs) == 0 {
		return nil, derp.BadRequest(location, "Must have at least one PrivilegeID")
	}

	criteria := exp.In("privilegeId", privilegeIDs)

	return service.Query(session, criteria)
}

// LoadByToken returns a single `Stream` that matches a particular `Token`
func (service *Stream) LoadByToken(session data.Session, token string, result *model.Stream) error {

	// If the token looks like an ObjectID, then try Load by ID first.
	if streamID, err := primitive.ObjectIDFromHex(token); err == nil {
		if err := service.LoadByID(session, streamID, result); err == nil {
			return nil
		}
	}

	// Default to Load by Token
	return service.Load(session, exp.Equal("token", token), result)
}

// LoadByID returns a single `Stream` that matches the provided streamID
func (service *Stream) LoadByID(session data.Session, streamID primitive.ObjectID, result *model.Stream) error {

	const location = "service.Stream.LoadByID"

	// RULE: StreamID is required
	if streamID.IsZero() {
		return derp.BadRequest(location, "StreamID is required")
	}

	return service.Load(session, exp.Equal("_id", streamID), result)
}

// LoadByURL returns a single `Stream` that matches the provided URL
func (service *Stream) LoadByURL(session data.Session, streamURL string, result *model.Stream) error {

	const location = "service.Stream.LoadByURL"

	// RULE: StreamURL is required
	if streamURL == "" {
		return derp.BadRequest(location, "StreamURL is required")
	}

	// Verify we have a valid URL
	uri, err := url.Parse(streamURL)

	if err != nil {
		return derp.Wrap(err, location, "Invalid URL", streamURL)
	}

	// Retrieve the Token from the request path
	token, _, err := service.ParsePath(uri)

	if err != nil {
		return derp.Wrap(err, location, "Invalid URL", streamURL)
	}

	return service.LoadByToken(session, token, result)
}

// LoadNavigationByID locates a single stream in the top level of the site hierarchy
func (service *Stream) LoadNavigationByID(session data.Session, streamID primitive.ObjectID, result *model.Stream) error {

	const location = "service.Stream.LoadNavigationByID"

	// RULE: StreamID is required
	if streamID.IsZero() {
		return derp.BadRequest(location, "StreamID is required")
	}

	criteria := exp.
		Equal("_id", streamID).
		AndEqual("parentId", primitive.NilObjectID)

	return service.Load(session, criteria, result)
}

func (service *Stream) LoadWithOptions(session data.Session, criteria exp.Expression, result *model.Stream, options ...option.Option) error {

	const location = "service.stream.LoadWithOptions"

	it, err := service.List(session, criteria, options...)

	if err != nil {
		return derp.Wrap(err, location, "Error getting iterator")
	}

	for it.Next(result) {
		return nil
	}

	return derp.NotFound(location, "collection is empty")
}

func (service *Stream) LoadFirstSibling(session data.Session, parentID primitive.ObjectID, result *model.Stream) error {
	return service.LoadWithOptions(session, exp.Equal("parentId", parentID), result, option.SortAsc("rank"))
}

func (service *Stream) LoadPrevSibling(session data.Session, parentID primitive.ObjectID, rank int, result *model.Stream) error {

	const location = "service.stream.LoadPreviousSibling"

	if rank == 0 {
		return service.LoadLastSibling(session, parentID, result)
	}

	criteria := exp.Equal("parentId", parentID).AndLessThan("rank", rank)

	err := service.LoadWithOptions(session, criteria, result, option.SortDesc("rank"))

	if err == nil {
		return nil
	}

	if derp.IsNotFound(err) {
		return service.LoadLastSibling(session, parentID, result)
	}

	return derp.Wrap(err, location, "Unable to load Previous Sibling")
}

func (service *Stream) LoadNextSibling(session data.Session, parentID primitive.ObjectID, rank int, result *model.Stream) error {

	const location = "service.stream.LoadNextSibling"

	criteria := exp.Equal("parentId", parentID).AndGreaterThan("rank", rank)

	err := service.LoadWithOptions(session, criteria, result, option.SortAsc("rank"))

	if err == nil {
		return nil
	}

	if derp.IsNotFound(err) {
		return service.LoadFirstSibling(session, parentID, result)
	}

	return derp.Wrap(err, location, "Unable to load Next Sibling")
}

func (service *Stream) LoadLastSibling(session data.Session, parentID primitive.ObjectID, result *model.Stream) error {
	return service.LoadWithOptions(session, exp.Equal("parentId", parentID), result, option.SortDesc("rank"))
}

func (service *Stream) LoadFirstAttachment(session data.Session, streamID primitive.ObjectID) (model.Attachment, error) {
	return service.attachmentService.LoadFirstByObjectID(session, model.AttachmentObjectTypeStream, streamID)
}

// MaxRank returns the maximum rank of all children of a stream
func (service *Stream) MaxRank(session data.Session, parentID primitive.ObjectID) (int, error) {
	collection := service.collection(session)
	return queries.MaxRank(session.Context(), collection, parentID)
}

/******************************************
 * Initialization Actions
 ******************************************/

// SetLocationTop sets a Stream to be a top-level navigation item
func (service *Stream) SetLocationTop(template *model.Template, stream *model.Stream) error {

	// RULE: Template must be allowed in the Top
	if !template.CanBeContainedBy("top") {
		return derp.BadRequest("service.Stream.SetLocationTop", "Template cannot be contained by 'top'", template)
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
		return derp.Unauthorized(location, "User ID is required")
	}

	// RULE: Template must be allowed in the Outbox
	if !template.CanBeContainedBy("outbox") {
		return derp.BadRequest(location, "Template cannot be contained by 'outbox'", template)
	}

	// Set values in the Stream
	stream.TemplateID = template.TemplateID
	stream.NavigationID = "profile"
	stream.ParentID = userID
	stream.ParentIDs = make([]primitive.ObjectID, 0)
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
		return derp.BadRequest(location, "Template cannot be contained by parent", template, parent)
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

// Shuffle assigns a unique random number to the "shuffle" field of each Stream
func (service *Stream) Shuffle(session data.Session) error {

	collection := service.collection(session)
	if err := queries.Shuffle(session.Context(), collection); err != nil {
		return derp.Wrap(err, "service.Stream.Shuffle", "Unable to shuffle users")
	}

	return nil
}

// SetAttributedTo assigns a User to the "attributedTo" field of each Stream
func (service *Stream) SetAttributedTo(user *model.User) {

	const location = "service.Stream.SetAttributedTo"

	// This is called asynchronously, so create a new database session
	session, cancel, err := service.factory.Session(time.Minute)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Unable to create database session"))
		return
	}

	defer cancel()

	collection := service.collection(session)

	if err := queries.SetAttributedTo(session.Context(), collection, user.PersonLink()); err != nil {
		derp.Report(derp.Wrap(err, location, "Unable to set attributedTo"))
	}
}

// DeleteByParent deletes all streams that are children of the provided parentID
func (service *Stream) DeleteByParent(session data.Session, parentID primitive.ObjectID, note string) error {

	// RULE: ParentID is required
	if parentID.IsZero() {
		return derp.Validation("ParentID cannot be zero")
	}

	return service.DeleteMany(session, exp.Equal("parentId", parentID), note)
}

// Delete RelatedDuplicate hard deletes any inbox/outbox streams that point to the same original.
func (service *Stream) DeleteRelatedDuplicate(session data.Session, parentID primitive.ObjectID, originalStreamID primitive.ObjectID) error {

	criteria := exp.Equal("parentId", parentID).AndEqual("data.originalStreamId", originalStreamID)

	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, "service.Stream.DeleteRelatedDuplicate", "Unable to delete related duplicate")
	}

	return nil
}

// MapByPrivileges returns a map of PrivilegeIDs to a slice of StreamIDs that grant additional access
// to Identities that hold of that Privileges.
func (service *Stream) MapByPrivileges(session data.Session, privileges ...model.Privilege) (map[primitive.ObjectID][]primitive.ObjectID, error) {

	const location = "service.Stream.MapByPrivileges"

	// RULE: If no privileges are provided, then return an empty map
	if len(privileges) == 0 {
		return make(mapof.Slices[primitive.ObjectID, primitive.ObjectID]), nil
	}

	// Scan all privileges for CircleIDs and MerchantAccounts/RemoteProductIDs
	privilegeIDs := make([]primitive.ObjectID, 0, len(privileges))

	for _, privilege := range privileges {

		if !privilege.CircleID.IsZero() {
			privilegeIDs = append(privilegeIDs, privilege.CircleID)
		}

		if !privilege.ProductID.IsZero() {
			privilegeIDs = append(privilegeIDs, privilege.ProductID)
		}
	}

	// RULE: If no CircleIDs or ProductIDs are defined, then return an empty map
	if len(privilegeIDs) == 0 {
		return make(mapof.Slices[primitive.ObjectID, primitive.ObjectID]), nil
	}

	// Find all Streams that match the included privilegeIDs
	streams, err := service.RangeByPrivileges(session, privilegeIDs...)

	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to load streams", privilegeIDs)
	}

	// Translate the range of Streams into a map of privilegeID => streamIDs
	result := make(mapof.Slices[primitive.ObjectID, primitive.ObjectID])

	for stream := range streams {
		for _, privilegeID := range stream.PrivilegeIDs {
			result.Add(privilegeID, stream.StreamID)
		}
	}

	// Ugly, but she rides.
	return result, nil
}

// ParsePathextracts the Stream token and actionID from a URL
func (service *Stream) ParsePath(uri *url.URL) (string, string, error) {

	// Verify the URL matches this service
	if dt.AddProtocol(uri.Host) != service.host {
		return "", "", derp.BadRequest("service.Stream.LoadByURL", "Hostname must match this server", uri.String())
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
func (service *Stream) ParseURL(session data.Session, streamURL string) (primitive.ObjectID, error) {

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
	if err := service.LoadByToken(session, path, &stream); err != nil {
		return primitive.NilObjectID, derp.Wrap(err, location, "Invalid Token", path)
	}

	return stream.StreamID, nil
}

// calcParentIDs scans the parent chain of a stream and generates a "breadcrumbs" slice
// of all of this Stream's parents
func (service *Stream) calcParentIDs(session data.Session, stream *model.Stream) {

	// If this stream has no parent, then it has no parent IDs
	if stream.ParentID == primitive.NilObjectID {
		stream.ParentIDs = id.NewSlice()
		return
	}

	// Otherwise, load the Parent stream and try to use its parentIDs
	maybeParentStream := model.NewStream()
	if err := service.LoadByID(session, stream.ParentID, &maybeParentStream); err == nil {
		stream.ParentIDs = append(maybeParentStream.ParentIDs, stream.ParentID)
		return
	}

	// Fall through: Just use the Parent (probably a User)
	stream.ParentIDs = []primitive.ObjectID{stream.ParentID}
}

func (service *Stream) calcDefaultAllow(template *model.Template, stream *model.Stream) {

	// NILCHECK: Template cannot be empty
	if template == nil {
		return
	}

	// NILCHECK: Stream cannot be empty
	if stream == nil {
		return
	}

	// Find the default action/roles for this Stream
	defaultAction := template.Default()
	defaultRoles := defaultAction.AllowedRoles(stream.StateID)

	// Calculate the GroupIDs and PrivilegeIDs for these roles
	groupIDs := stream.RolesToGroupIDs(defaultRoles...)
	privilegeIDs := stream.RolesToPrivilegeIDs(defaultRoles...)

	// Update the Stream wtih the calculated values
	result := append(groupIDs, privilegeIDs...)
	result = result.Compact()
	stream.DefaultAllow = result
}

// CalcContext calculates the conversational context for a given stream,
// IF it can be determined.
func (service *Stream) calcContext(stream *model.Stream) {

	// If this is an original stream (not a reply) then its context is itself.
	if stream.InReplyTo == "" {
		stream.Context = stream.ActivityPubURL()
		return
	}

	// Load the "InReplyTo" document from the ActivityStream and use its
	// context.  Note: this should have been calculated already via th
	// ascontextmaker client.
	activityService := service.factory.ActivityStream(model.ActorTypeStream, stream.StreamID)
	document, _ := activityService.Client().Load(stream.InReplyTo)

	if context := document.Context(); context != "" {
		stream.Context = document.Context()
		return
	}

	// If a context could not be assigned, then use the InReplyTo value instead.
	stream.Context = stream.InReplyTo
}

// CalcPrivileges denormalizes all privileges (CircleIDs and ProductIDs)
// for a Stream into a single data structure that can be scanned
// easily by MongoDB.
func (service *Stream) calcPrivilegeIDs(stream *model.Stream) {
	circles := flatten(stream.Circles)
	privileges := flatten(stream.Products)
	stream.PrivilegeIDs = model.Permissions(append(circles, privileges...))
}

func (service *Stream) CalculateTags(session data.Session, stream *model.Stream) {

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
	hashtagNames, _, err := service.searchTagService.NormalizeTags(session, hashtags...)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Error normalizing tags"))
	}

	// Apply the #hashtags back to the Stream
	stream.Hashtags = hashtagNames
}

func (service *Stream) NotifyInReplyTo(session data.Session, inReplyTo string) {

	const location = "service.Following.notifyInReplyTo"

	// If this is not a reply, then skip
	if inReplyTo == "" {
		return
	}

	// If the "inReplyTo" is not on this server, then skip
	if !strings.HasPrefix(inReplyTo, service.host) {
		return
	}

	inReplyTo, _ = strings.CutPrefix(inReplyTo, service.host)

	// Get the 'token' part of the URL
	_, token, _ := strings.Cut(inReplyTo, "/")

	stream := model.NewStream()
	if err := service.LoadByToken(session, token, &stream); err != nil {

		derp.Report(derp.Wrap(err, location, "Unable to locate 'InReplyTo' stream", inReplyTo))
		// If the "inReplyTo" stream cannot be loaded, then log
		// the error but do not fail the rest of the transaction
		return
	}

	// Notify the `inReplyTo` stream
	service.sseUpdateChannel <- realtime.NewMessage_NewReplies(stream.StreamID)

	// Glory to Rome.
}

/******************************************
 * Migration Methods
 ******************************************/

// Move locates all Streams inside the profile of the provided UserID, and moves them
// using the 'movedTo' forwarding address
func (service *Stream) MoveByUserID(session data.Session, userID primitive.ObjectID, movedTo string) error {

	const location = "service.Stream.MoveByUserID"

	// Range over all Streams that match this User
	streams, err := service.RangeByParentIDs(session, userID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to query streams", userID)
	}

	// Move each stream one-by-one
	for stream := range streams {

		if err := service.Move(session, &stream, movedTo); err != nil {
			return derp.Wrap(err, location, "Unable to move Stream", stream)
		}
	}

	// Success!
	return nil
}

// Move updates a Stream to indicate that it has been moved to another server,
// and deletes all related Attachments and Mentions.
func (service *Stream) Move(session data.Session, stream *model.Stream, movedTo string) error {

	const location = "service.Stream.Move"

	// Set the `MovedTo` value to forward to the Oracle on the new server
	stream.MovedTo = movedTo

	// Zero out (almost) all other fields in this stream
	stream.TemplateID = ""
	stream.ParentTemplateID = ""
	stream.StateID = ""
	stream.SocialRole = ""
	stream.Groups = mapof.NewObject[id.Slice]()
	stream.Circles = mapof.NewObject[id.Slice]()
	stream.Products = mapof.NewObject[id.Slice]()
	stream.PrivilegeIDs = model.NewPermissions()
	stream.DefaultAllow = model.Permissions{model.MagicGroupIDAnonymous}
	stream.Label = ""
	stream.Summary = ""
	stream.Icon = ""
	stream.IconURL = ""
	stream.Context = ""
	stream.InReplyTo = ""
	stream.Content = model.NewContent()
	stream.Widgets = set.NewSlice[model.StreamWidget]()
	stream.Hashtags = sliceof.NewString()
	stream.Location = geo.NewAddress()
	stream.Data = mapof.NewAny()
	stream.StartDate = datetime.New()
	stream.EndDate = datetime.New()
	stream.Syndication = delta.NewSlice[string]()
	stream.Shuffle = 0
	stream.UnPublishDate = time.Now().Unix()
	stream.IsFeatured = false
	stream.IsSubscribable = false

	// Keep these original values
	// stream.URL
	// stream.Token
	// stream.AttributedTo
	// stream.PublishDate

	// Update the Stream with the new "movedTo" value but skip all other business rules.
	if err := service.collection(session).Save(stream, "moved"); err != nil {
		return derp.Wrap(err, location, "Unable to save Stream")
	}

	// Delete any related Attachments
	if err := service.attachmentService.DeleteByCriteria(session, "Stream", stream.StreamID, exp.All(), "moved"); err != nil {
		return derp.Wrap(err, location, "Unable to delete Attachments")
	}

	// Delete any related Mentions
	if err := service.mentionService.DeleteByObjectID(session, model.MentionTypeStream, stream.StreamID, "moved"); err != nil {
		return derp.Wrap(err, location, "Unable to delete Mentions")
	}

	return nil
}

/******************************************
 * SearchResulter Interface
 ******************************************/

// SearchResult returns a SearchResult object that represents this Stream in the search index
func (service *Stream) SearchResult(stream *model.Stream) model.SearchResult {

	result := model.NewSearchResult()

	// If the stream has been published, then try to generate a SearchResult for it.
	if stream.IsPublished() {

		// Only create a search result if the stream is visible by ALL people
		if stream.DefaultAllowAnonymous() {

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
					result.Local = true

					if stream.Location.NotZero() {
						result.Location = stream.Location.GeoPoint()
					}

					if tagString := template.SearchOptions.Execute("tags", stream); tagString != "" {
						tags := strings.Split(tagString, " ")
						result.Tags = append(result.Tags, tags...)
					}

					return result
				}
			}
		}
	}

	// Fall through means this Stream is not searchable
	result.URL = stream.URL
	result.DeleteDate = time.Now().Unix()
	return result
}

// Hostname returns the hostname (domain only) for this service
func (service *Stream) Hostname() string {
	return dt.NameOnly(service.host)
}
