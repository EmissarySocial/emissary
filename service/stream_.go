package service

import (
	"context"
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
	"github.com/benpate/exp"
	"github.com/benpate/geo"
	"github.com/benpate/hannibal/vocab"
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
	"github.com/benpate/uri"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Stream manages all interactions with the Stream collection
type Stream struct {
	activityService     *ActivityStream
	attachmentService   *Attachment
	circleService       *Circle
	collectionService   *Collection
	contentService      *Content
	domainService       *Domain
	draftService        *StreamDraft
	geocodeService      GeocodeAddress
	importService       *Import
	importItemService   *ImportItem
	keyService          *EncryptionKey
	locatorService      *Locator
	notificationService *Notification
	outboxService       *Outbox
	permissionService   *Permission
	searchTagService    *SearchTag
	templateService     *Template
	followerService     *Follower
	ruleService         *Rule
	userService         *User
	webhookService      *Webhook
	host                string
	mediaserver         mediaserver.MediaServer
	queue               *queue.Queue
	sseUpdateChannel    chan<- realtime.Message
	newSession          func(timeout time.Duration) (data.Session, context.CancelFunc, error)
}

// NewStream returns a fully populated Stream service.
func NewStream() Stream {
	return Stream{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Stream) Refresh(factory *Factory) {
	service.activityService = factory.ActivityStream()
	service.attachmentService = factory.Attachment()
	service.circleService = factory.Circle()
	service.collectionService = factory.Collection()
	service.contentService = factory.Content()
	service.domainService = factory.Domain()
	service.draftService = factory.StreamDraft()
	service.followerService = factory.Follower()
	service.geocodeService = factory.GeocodeAddress()
	service.importService = factory.Import()
	service.importItemService = factory.ImportItem()
	service.keyService = factory.EncryptionKey()
	service.locatorService = factory.Locator()
	service.notificationService = factory.Notification()
	service.outboxService = factory.Outbox()
	service.permissionService = factory.Permission()
	service.ruleService = factory.Rule()
	service.searchTagService = factory.SearchTag()
	service.templateService = factory.Template()
	service.userService = factory.User()
	service.webhookService = factory.Webhook()
	service.mediaserver = factory.MediaServer()
	service.queue = factory.Queue()
	service.newSession = factory.Session

	service.host = factory.Host()
	service.sseUpdateChannel = factory.SSEUpdateChannel()
}

func (service *Stream) Startup(session data.Session, theme *model.Theme) error {

	const location = "service.Stream.Startup"

	// Try to count the number of streams currently in the database
	count, err := service.Count(session, exp.All())

	if err != nil {
		return derp.Wrap(err, location, "Counting streams")
	}

	// If the database is not empty, then do not add more...
	if count > 0 {
		return nil
	}

	for _, data := range theme.StartupStreams {

		// Build a Stream from the theme's startup data
		stream, err := service.newStartupStream(data)

		if err != nil {
			return derp.Wrap(err, location, "Building startup stream", data)
		}

		// Save the new Stream to the database.  Save is the single validation gate: it
		// loads the template, normalizes the value against the template schema (which
		// sanitizes content, clamps lengths, etc.), and only then persists.  Startup must
		// NOT pre-validate, because Validate rejects any value that would be rewritten --
		// and freshly-rendered article HTML is always rewritten by the "html" sanitizer.
		if err := service.Save(session, &stream, "Created by Startup"); err != nil {
			return derp.Wrap(err, location, "Saving startup stream", stream)
		}
	}

	return nil
}

// newStartupStream builds a single published Stream from one theme.StartupStreams
// entry.  It is separated from Startup so the schema-application step -- the part that
// silently failed for every startup stream -- can be exercised directly by tests without
// a database, template registry, or webhook service.  The returned Stream is NOT yet
// normalized or persisted; Startup passes it to Save, which is the single validation gate.
func (service *Stream) newStartupStream(data mapof.Any) (model.Stream, error) {

	const location = "service.Stream.newStartupStream"

	stream := model.NewStream()

	// Apply the theme's scalar values.  The "content" key holds a whole model.Content
	// object, which rosetta's object-Set cannot assign in a single call (it only accepts
	// dotted paths, e.g. "content.html"), so it is excluded here and applied directly
	// below.  Copy the map instead of deleting the key, because themes are cached and
	// shared across requests.
	values := make(mapof.Any, len(data))
	for key, value := range data {
		if key == "content" {
			continue
		}
		values[key] = value
	}

	if err := service.Schema().SetAll(&stream, values); err != nil {
		return stream, derp.Wrap(err, location, "Setting stream data", values)
	}

	// Set this Stream as "Published"
	stream.PublishDate = 0

	// If we have default content, then add that too.
	if content, ok := data["content"].(model.Content); ok {
		stream.Content = content
	}

	return stream, nil
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
		return nil, derp.Wrap(err, "service.Stream.Range", "Creating iterator", criteria)
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
		return nil, derp.Wrap(err, location, "Creating iterator", criteria)
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
		return derp.Wrap(err, location, "Loading Stream", criteria)
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
			return derp.Wrap(err, location, "Calculating max rank")
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
			return derp.Wrap(err, location, "Geocoding stream", stream.Location)
		}
	}

	// RULE: Every Stream must be associated with a Template
	if stream.TemplateID == "" {
		return derp.BadRequest(location, "Stream cannot be saved without a TemplateID", stream)
	}

	// Load the template used by this Stream
	template, err := service.templateService.Load(stream.TemplateID)

	if err != nil {
		return derp.Wrap(err, location, "Loading template", stream.TemplateID)
	}

	// Copy default values from the Template
	stream.SocialRole = template.SocialRole
	stream.IsSubscribable = template.IsSubscribable()
	stream.URL = service.host + "/" + stream.StreamID.Hex()

	// RULE: Calculate "defaultAllow" groups for this stream.
	service.calcDefaultAllow(&template, stream)

	// RULE: Extract and linkify #hashtags for Templates that configure tagging.  This runs
	// BEFORE Normalize so the injected anchors pass through the same schema sanitization as
	// any other content HTML.  @mentions are extracted from the same paths, but only as far
	// as their handles -- resolving a handle to an Actor URL is a network call, so it happens
	// once at publish time (resolveMentions) rather than on every save.
	if len(template.TagPaths) > 0 {
		service.CalculateTags(session, stream)
		service.CalculateMentions(stream)
		service.applyHashtagLinks(&template, stream)
	}

	// Normalize the value (using the template-specific schema) before saving.  Values are
	// rewritten in place to conform to the schema, so that legacy data written under older
	// rules is repaired progressively as records are saved.  The template schema inherits
	// the full Stream schema as its baseline, so this covers every Stream property while
	// honoring the template's format overrides.
	rewrites, err := template.Schema.Normalize(stream)

	if err != nil {
		return derp.Wrap(err, location, "Invalid Stream: using TemplateSchema", stream)
	}

	if len(rewrites) > 0 {
		log.Debug().Strs("rewrites", rewrites).Str("streamId", stream.StreamID.Hex()).Msg("Stream values normalized during save")
	}

	// RULE: calculate Parent IDs
	service.calcParentIDs(session, stream)

	// RULE: Calculate privileges for this stream
	service.calcPrivilegeIDs(stream)

	// Try to save the Stream to the database
	if err := service.collection(session).Save(stream, note); err != nil {
		return derp.Wrap(err, location, "Saving Stream", stream, note)
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
		return derp.Wrap(err, location, "Deleting Stream")
	}

	return nil
}

// Delete removes an Stream from the database (virtual delete)
func (service *Stream) Delete(session data.Session, stream *model.Stream, note string) error {

	const location = "service.Stream.Delete"

	// Delete this Stream
	if err := service.collection(session).Delete(stream, note); err != nil {
		return derp.Wrap(err, location, "Deleting Stream from database", stream, note)
	}

	// Send Webhooks (if configured)
	service.webhookService.Send(stream, model.WebhookEventStreamDelete)

	if stream.IsPublished() {
		service.webhookService.Send(stream, model.WebhookEventStreamPublishUndo)

		if err := service.sendSyndicationMessages(session, stream, nil, nil, stream.Syndication.Values); err != nil {
			derp.Report(derp.Wrap(err, location, "Sending syndication messages", stream))
		}
	}

	// RULE: Delete all related Children
	if err := service.DeleteByParent(session, stream.StreamID, note); err != nil {
		derp.Report(derp.Wrap(err, location, "Deleting child streams", stream, note))
	}

	// RULE: Delete all related Attachments
	if err := service.attachmentService.DeleteAll(session, model.AttachmentObjectTypeStream, stream.StreamID, note); err != nil {
		derp.Report(derp.Wrap(err, location, "Deleting attachments", stream, note))
	}

	// RULE: Delete all related Drafts
	if err := service.draftService.Delete(session, stream, note); err != nil {
		derp.Report(derp.Wrap(err, location, "Deleting drafts", stream, note))
	}

	// RULE: Delete related Context Collection (if exists)
	if err := service.collectionService.DeleteByURL(session, stream.Context); err != nil {
		derp.Report(derp.Wrap(err, location, "Deleting context collection", stream, note))
	}

	// RULE: If this Stream is a reply, remove it from its parent's Replies collection (and refresh the count).
	if err := service.RemoveReply(session, stream.InReplyTo, stream.ActivityPubURL()); err != nil {
		derp.Report(derp.Wrap(err, location, "Removing reply from parent collection", stream, note))
	}

	// RULE: Delete Outbox Messages
	if err := service.outboxService.DeleteByParentID(session, model.FollowerTypeStream, stream.StreamID); err != nil {
		derp.Report(derp.Wrap(err, location, "Deleting outbox messages", stream, note))
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
		return derp.Wrap(err, location, "Listing streams to delete", criteria)
	}

	for stream := model.NewStream(); it.Next(&stream); stream = model.NewStream() {
		if err := service.Delete(session, &stream, note); err != nil {
			return derp.Wrap(err, location, "Deleting stream", stream)
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

// RangePublishedByParent returns an iterator over the currently-published child streams of the
// provided parentID. It applies the publish-date window (a Stream's own lifecycle state, not a
// permission check) in the query so unpublished/scheduled children never leave the database.
// Per-viewer permission filtering (Stream.IsVisibleTo) is intentionally left to the caller.
func (service *Stream) RangePublishedByParent(session data.Session, parentID primitive.ObjectID) (iter.Seq[model.Stream], error) {

	now := time.Now().Unix()

	criteria := exp.Equal("parentId", parentID).
		AndLessOrEqual("publishDate", now).
		AndGreaterOrEqual("unpublishDate", now)

	return service.Range(session, criteria)
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

	// Retrieve the Stream token from the request URL
	token, _, err := service.locatorService.ParseStream(streamURL)

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
		return derp.Wrap(err, location, "Getting iterator")
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

	return derp.Wrap(err, location, "Loading Previous Sibling")
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

	return derp.Wrap(err, location, "Loading Next Sibling")
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
		return derp.Wrap(err, "service.Stream.Shuffle", "Shuffling users")
	}

	return nil
}

// SetAttributedTo assigns a User to the "attributedTo" field of each Stream
func (service *Stream) SetAttributedTo(user *model.User) {

	const location = "service.Stream.SetAttributedTo"

	// This is called asynchronously, so create a new database session
	session, cancel, err := service.newSession(time.Minute)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Creating database session"))
		return
	}

	defer cancel()

	collection := service.collection(session)

	if err := queries.SetAttributedTo(session.Context(), collection, user.PersonLink()); err != nil {
		derp.Report(derp.Wrap(err, location, "Setting attributedTo"))
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
		return derp.Wrap(err, "service.Stream.DeleteRelatedDuplicate", "Deleting related duplicate")
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
		return nil, derp.Wrap(err, location, "Loading streams", privilegeIDs)
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

// ParsePath extracts the Stream token and actionID from a URL
func (service *Stream) ParsePath(parsedURL *url.URL) (string, string, error) {

	const location = "service.Stream.ParsePath"

	// Verify the URL matches this service
	if uri.PrependProtocol(parsedURL.Host) != service.host {
		return "", "", derp.BadRequest(location, "Hostname must match this server", parsedURL.String())
	}

	// Load the Stream using the token
	path := list.BySlash(strings.TrimPrefix(parsedURL.Path, "/"))
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
		derp.Report(derp.Wrap(err, location, "Loading Template", stream.TemplateID))
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
		derp.Report(derp.Wrap(err, location, "Normalizing tags"))
	}

	// Apply the #hashtags back to the Stream.
	//
	// Hashtags is DEPRECATED and dual-written for one release while the external template
	// packages migrate (see projects/TAGS-UNIFICATION.md).  Tags is the value readers use.
	// Only Hashtag-typed entries are replaced, so @mentions survive untouched.
	stream.Hashtags = hashtagNames

	hashtagTags := make(model.TagList, 0, len(hashtagNames))

	for _, name := range hashtagNames {
		hashtagTags = append(hashtagTags, model.NewTag(vocab.LinkTypeHashtag, name))
	}

	stream.Tags = model.ReplaceTagsOfType(stream.Tags, vocab.LinkTypeHashtag, hashtagTags)
}

// mentionResolveTimeout bounds an ENTIRE batch of @mention lookups.  `remote` allows a full
// minute per request and each resolution is two requests, so without this a single blackholed
// host could stall a user's save for minutes.
const mentionResolveTimeout = 5 * time.Second

// mentionResolveConcurrency caps simultaneous @mention lookups, so a post naming a large number
// of people cannot open an unbounded number of outbound connections at once.
const mentionResolveConcurrency = 8

// CalculateMentions scans the Template-defined TagPaths on this Stream for @mentions and
// merges them into the Stream's Tags, resolving any handle it has not seen before.
func (service *Stream) CalculateMentions(stream *model.Stream) {

	// Extraction is pure string work.  Resolution is not -- it is a WebFinger lookup followed by
	// an Actor fetch -- but it happens at most once per handle: a handle already present keeps
	// its Href, and `ascache` keys actor documents by the handle string, so a handle first seen
	// on some OTHER Stream is a cache hit here.  Only genuinely new handles reach the network.
	// See projects/TAGS-UNIFICATION.md and bugs/BUG-09-Mentions-Not-Emitted.md.

	const location = "service.Stream.CalculateMentions"

	// Load the template (to get the tag paths)
	template, err := service.templateService.Load(stream.TemplateID)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Loading Template", stream.TemplateID))
		return
	}

	// Index the Hrefs already known, so that re-scanning preserves them.  This carries the
	// TagHrefUnresolvable sentinel forward too, which is what stops a handle that cannot be
	// resolved from being looked up again on every single save.
	known := make(map[string]string, len(stream.Tags))

	for _, tag := range model.TagsOfType(stream.Tags, vocab.LinkTypeMention) {
		known[tag.Name] = tag.Href
	}

	// Scan each tag path in the Stream for @mentions
	schema := service.Schema()
	mentions := make(model.TagList, 0, len(stream.Tags))
	seen := make(map[string]struct{}, len(stream.Tags))
	hostname := uri.Hostname(service.host)

	for _, path := range template.TagPaths {

		if value, err := schema.Get(stream, path); err == nil {

			// Massage the value into a cleanly scannable string
			stringValue := convert.String(value)
			stringValue = html.ToSearchText(stringValue)

			for _, handle := range parse.Mentions(stringValue) {

				// RULE: A bare "@" yields an empty token, which is not a handle.  Without this,
				// resolution would spend a WebFinger lookup on the empty string.
				if handle == "" {
					continue
				}

				// RULE: A handle with no hostname is anchored to THIS server -- on bandwagon.fm,
				// "@bob" means "@bob@bandwagon.fm".  Qualifying at extraction (rather than at
				// resolution) stores a handle that is unambiguous to the remote servers that read
				// this document, and makes "@bob" and "@bob@bandwagon.fm" in the same document
				// dedupe to a single Tag.
				if !strings.Contains(handle, "@") {

					// With no hostname to anchor to, the handle can never resolve. Drop it here
					// rather than storing an address that is guaranteed to fail on every publish.
					if hostname == "" {
						continue
					}

					handle = handle + "@" + hostname
				}

				// RULE: One handle may be mentioned many times in a single document
				if _, exists := seen[handle]; exists {
					continue
				}

				seen[handle] = struct{}{}

				mentions = append(mentions, model.Tag{
					Type: vocab.LinkTypeMention,
					Name: handle,
					Href: known[handle], // empty for handles that have never been looked up
				})
			}
		}
	}

	// Look up the Actor URL for every handle we have not seen before
	service.resolveMentions(mentions)

	// Apply the @mentions back to the Stream, leaving #hashtags untouched
	stream.Tags = model.ReplaceTagsOfType(stream.Tags, vocab.LinkTypeMention, mentions)
}

// resolveMentions fills in the Actor URL for every Mention Tag that does not already have one,
// rewriting the provided slice in place.
func (service *Stream) resolveMentions(tags model.TagList) {

	const location = "service.Stream.resolveMentions"

	// Collect the entries that still need a lookup.  Usually there are none.
	pending := make([]int, 0, len(tags))

	for index, tag := range tags {
		if tag.NeedsResolution() {
			pending = append(pending, index)
		}
	}

	if len(pending) == 0 {
		return
	}

	// Lookups are independent of each other, so they run concurrently: the cost of a post that
	// mentions twenty people tracks the SLOWEST handle rather than the sum of all twenty.  The
	// deadline bounds the whole batch, because `remote` allows a full minute PER REQUEST and
	// resolution is two requests -- far too long to hold a user's save.
	ctx, cancel := context.WithTimeout(context.Background(), mentionResolveTimeout)
	defer cancel()

	// Buffered to len(pending) so a straggler can always deliver its result and exit, even after
	// this function has stopped listening.  Nothing writes to `tags` except the loop below, so
	// abandoning a slow lookup cannot race with the caller.
	type resolved struct {
		index int
		href  string
	}

	results := make(chan resolved, len(pending))
	semaphore := make(chan struct{}, mentionResolveConcurrency)

	for _, index := range pending {

		go func(index int) {

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			handle := tags[index].Name
			actor, err := service.activityService.GetActor(handle)

			if err != nil {
				derp.Report(derp.Wrap(err, location, "Unable to resolve @mention; publishing without it", handle))
				results <- resolved{index: index, href: model.TagHrefUnresolvable}
				return
			}

			if actorID := actor.ID(); actorID != "" {
				results <- resolved{index: index, href: actorID}
				return
			}

			results <- resolved{index: index, href: model.TagHrefUnresolvable}
		}(index)
	}

	for range pending {

		select {

		case result := <-results:
			tags[result.index].Href = result.href

		case <-ctx.Done():

			// The batch blew its time budget.  Whatever resolved is kept; the rest keep an empty
			// Href, so the next save retries them rather than marking them permanently bad.
			derp.Report(derp.Wrap(ctx.Err(), location, "Timed out resolving @mentions; unresolved handles will retry on next save"))
			return
		}
	}
}

// applyHashtagLinks wraps each of the Stream's #hashtags in its content with a link to the
// Template's TagURL.  The links are absolute, because this content is read by other servers.
func (service *Stream) applyHashtagLinks(template *model.Template, stream *model.Stream) {

	// RULE: Only linkify when the Template defines a tag URL
	tagURL := model.HashtagURLPrefix(service.host, template.TagURL)

	if tagURL == "" {
		return
	}

	// RULE: Nothing to link if there are no hashtags
	hashtags := model.TagNames(stream.Tags, vocab.LinkTypeHashtag)

	if len(hashtags) == 0 {
		return
	}

	service.contentService.ApplyTags(&stream.Content, tagURL, hashtags)
}

// NotifyInReplyTo sends an SSE notification to any stream that is referenced in the "inReplyTo" field of a Stream
func (service *Stream) NotifyInReplyTo(session data.Session, inReplyTo string) {

	const location = "service.Stream.notifyInReplyTo"

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

		derp.Report(derp.Wrap(err, location, "Locating 'InReplyTo' stream", inReplyTo))
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
		return derp.Wrap(err, location, "Querying streams", userID)
	}

	// Move each stream one-by-one
	for stream := range streams {

		if err := service.Move(session, &stream, movedTo); err != nil {
			return derp.Wrap(err, location, "Moving Stream", stream)
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
	stream.Tags = model.NewTagList()
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
		return derp.Wrap(err, location, "Saving Stream")
	}

	// Delete any related Attachments
	if err := service.attachmentService.DeleteByCriteria(session, "Stream", stream.StreamID, exp.All(), "moved"); err != nil {
		return derp.Wrap(err, location, "Deleting Attachments")
	}

	// Delete any related Notifications (mentions/replies/reactions that referenced this Stream)
	if err := service.notificationService.DeleteByStreamID(session, stream.StreamID, "moved"); err != nil {
		return derp.Wrap(err, location, "Deleting Notifications")
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
					result.Tags = slice.Map(model.TagNames(stream.Tags, vocab.LinkTypeHashtag), model.ToToken)
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
	return uri.Hostname(service.host)
}
