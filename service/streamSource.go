package service

import (
	"iter"
	"net/url"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/realtime"
	"github.com/EmissarySocial/emissary/service/content"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TaskSyncStreamSource is the BACKGROUND queue task that synchronizes one StreamSource record.
// Its usual trigger is an unauthenticated webhook that fans out to every record sharing a token.
const TaskSyncStreamSource = "SyncStreamSource"

// TaskSyncStreamSourceNow is the INTERACTIVE queue task behind the Sync Now button.  It runs the
// same handler, and reports through the same lifecycle hooks, as TaskSyncStreamSource: only its
// priority and its lack of a signature differ.
const TaskSyncStreamSourceNow = "SyncStreamSourceNow"

// IsSyncStreamSourceTask returns TRUE for either synchronization task.
//
// RULE: Every consumer-side registration must accept BOTH names -- the dispatch switch and all
// three lifecycle hooks.  A hook that tested only one name would silently stop recording status
// for the other path, and nothing reports a hook that declined a task.
func IsSyncStreamSourceTask(name string) bool {
	return (name == TaskSyncStreamSource) || (name == TaskSyncStreamSourceNow)
}

// streamWriter is the slice of the Stream service that a synchronization uses.  The interface is
// declared here, beside its caller, so that a test can drive a sync without assembling the
// Template, Geocode, and Webhook services that Stream.Save depends on.
type streamWriter interface {
	LoadByID(session data.Session, streamID primitive.ObjectID, result *model.Stream) error
	ValidateToken(session data.Session, streamID primitive.ObjectID, token string) error
	Save(session data.Session, stream *model.Stream, note string) error
}

// StreamSource manages StreamSource records, which copy content from outside of Emissary into Streams
type StreamSource struct {
	streamService  streamWriter
	contentService *Content
	adapters       map[string]content.Adapter
	queue          *queue.Queue
	hostname       string
}

// NewStreamSource returns a fully initialized StreamSource service
func NewStreamSource() StreamSource {
	return StreamSource{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service
func (service *StreamSource) Refresh(factory *Factory) {

	service.streamService = factory.Stream()
	service.contentService = factory.Content()
	service.queue = factory.Queue()
	service.hostname = factory.Hostname()

	// One Adapter per Method, keyed by the token that the record stores.  AllowPrivateIPs is
	// FALSE in production, so the adapter refuses to connect to a private address.
	httpsAdapter := content.NewHTTPS(factory.ActivityStream().AllowPrivateIPs())

	service.adapters = map[string]content.Adapter{
		httpsAdapter.Protocol(): httpsAdapter,
	}
}

// Close stops any background processes controlled by this service
func (service *StreamSource) Close() {
	// Nothing to close.
}

/******************************************
 * Common Data Methods
 ******************************************/

// collection returns the StreamSource collection for the provided database session
func (service *StreamSource) collection(session data.Session) data.Collection {
	return session.Collection("StreamSource")
}

// New returns a newly initialized StreamSource record
func (service *StreamSource) New() model.StreamSource {
	return model.NewStreamSource()
}

// Count returns the number of StreamSource records that match the provided criteria
func (service *StreamSource) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// List returns an iterator of the StreamSource records that match the provided criteria
func (service *StreamSource) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc over the StreamSource records that match the provided criteria
func (service *StreamSource) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.StreamSource], error) {

	iterator, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.StreamSource.Range", "Creating iterator", criteria)
	}

	return RangeFunc(iterator, model.NewStreamSource), nil
}

// Load retrieves a StreamSource record from the database
func (service *StreamSource) Load(session data.Session, criteria exp.Expression, streamSource *model.StreamSource) error {

	if err := service.collection(session).Load(notDeleted(criteria), streamSource); err != nil {
		return derp.Wrap(err, "service.StreamSource.Load", "Loading StreamSource", criteria)
	}

	return nil
}

// Save adds or updates a StreamSource record, and queues a synchronization with its source
func (service *StreamSource) Save(session data.Session, streamSource *model.StreamSource, note string) error {

	const location = "service.StreamSource.Save"

	// RULE: Saving REACHES THE NETWORK.  Every save queues a sync, because nothing polls and a
	// save is the only moment a human tells Emissary this record is worth reading.  A repeat
	// costs one conditional GET that answers 304, so the cheap case is free -- but a caller that
	// saves MANY records in a loop fans out one outbound fetch per record, with nothing at the
	// call site to say so.  Bookkeeping writes call service.save and avoid all of this.

	// RULE: A StreamSource record is always attached to a Stream
	if streamSource.StreamID.IsZero() {
		return derp.BadRequest(location, "StreamSource must be attached to a Stream")
	}

	// RULE: The address is checked before the schema sees it, because the schema's own error quotes
	// the address, and an address can carry a password
	if err := validateStreamSourceURL(streamSource.URL); err != nil {
		return derp.Wrap(err, location, "Invalid URL", streamSource.StreamSourceID)
	}

	// RULE: The webhook token is editable by hand, so a short one is guessable and a pasted URL
	// is a token nothing can ever match -- the route matches one path segment.  The endpoint
	// re-checks the length, because a record saved by an older build could carry anything.
	if !model.IsValidWebhookToken(streamSource.WebhookToken()) {
		return derp.BadRequest(location, "Webhook token must be at least 16 letters, digits, dots, dashes, or underscores", model.StreamSourceWebhookTokenMinLength)
	}

	// Validate everything else against the schema
	if _, err := service.Schema().Validate(streamSource); err != nil {
		return derp.Wrap(err, location, "Validating StreamSource", streamSource.StreamSourceID)
	}

	// The status is written BEFORE the save, so a human watching the settings screen sees the sync
	// start rather than a record that looks untouched
	streamSource.Status = model.StreamSourceStatusLoading
	streamSource.StatusMessage = ""
	markChecked(streamSource)

	if err := service.save(session, streamSource, note); err != nil {
		return derp.Wrap(err, location, "Saving StreamSource", streamSource.StreamSourceID, note)
	}

	// Every caller of Save is a human at the settings screen -- the source form, or the Sync Now
	// button -- so this is the interactive task.  Background bookkeeping goes through service.save
	// and queues nothing at all.
	service.PublishSyncTaskNow(session, streamSource.StreamSourceID)

	return nil
}

// save writes a StreamSource record and nudges every browser watching its Stream.  EVERY write
// path goes through here, including the status bookkeeping that skips Save's validation: the
// settings screen does not poll, so a status it never hears about reads as no status at all.
func (service *StreamSource) save(session data.Session, streamSource *model.StreamSource, note string) error {

	// The caller wraps this with its own location, so it travels bare
	if err := service.collection(session).Save(streamSource, note); err != nil {
		return err
	}

	service.publishSSE(session, streamSource.StreamID)

	return nil
}

// publishSSE sends a best-effort realtime nudge to every browser watching this record's Stream.
// It rides the post-commit spool so a nudge cannot fire before the row it describes is committed,
// and queue.WithInline() keeps the task in this process, which is where the broker's sockets are.
//
// RULE: The message is addressed by the STREAM's ID, not the StreamSource's.  A StreamSource has
// no page and no SSE route of its own; the screen that cares is a Stream page.
func (service *StreamSource) publishSSE(session data.Session, streamID primitive.ObjectID) {

	postcommit.Publish(session, service.queue, "PublishRealtimeMessage", mapof.Any{
		"hostname": service.hostname,
		"objectId": streamID.Hex(),
		"topic":    realtime.TopicStreamSourceUpdated,
	}, queue.WithInline())
}

// Delete removes a StreamSource record from the database (virtual delete)
func (service *StreamSource) Delete(session data.Session, streamSource *model.StreamSource, note string) error {

	const location = "service.StreamSource.Delete"

	// RULE: A soft-deleted record keeps its webhook token for the length of the purge window, and
	// that is fine: the token only triggers a sync, and every query here filters on deleteDate, so
	// a ping arriving in that window matches nothing.
	if err := service.collection(session).Delete(streamSource, note); err != nil {
		return derp.Wrap(err, location, "Deleting StreamSource", streamSource.StreamSourceID, note)
	}

	// Gone, but not forgotten.  Well, forgotten in a week.
	return nil
}

// DeleteByStreamID removes every StreamSource record attached to a Stream (virtual delete)
func (service *StreamSource) DeleteByStreamID(session data.Session, streamID primitive.ObjectID, note string) error {

	const location = "service.StreamSource.DeleteByStreamID"

	// RULE: A zero StreamID is refused, because it matches every record whose Stream was never
	// set -- which selects records rather than naming one Stream's
	if streamID.IsZero() {
		return derp.BadRequest(location, "StreamID cannot be zero")
	}

	streamSources, err := service.Range(session, exp.Equal("streamId", streamID))

	if err != nil {
		return derp.Wrap(err, location, "Listing StreamSources", streamID)
	}

	// RULE: One failure does not strand the rest.  The Stream that reaches these records is
	// already gone, so whatever survives this loop is an orphan nothing will delete again.
	for streamSource := range streamSources {
		if err := service.Delete(session, &streamSource, note); err != nil {
			derp.Report(derp.Wrap(err, location, "Deleting StreamSource", streamSource.StreamSourceID, note))
		}
	}

	return nil
}

// Schema returns the rosetta schema that describes a StreamSource record
func (service *StreamSource) Schema() schema.Schema {
	return schema.New(model.StreamSourceSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

// LoadByID retrieves a StreamSource record by its unique identifier
func (service *StreamSource) LoadByID(session data.Session, streamSourceID primitive.ObjectID, streamSource *model.StreamSource) error {

	const location = "service.StreamSource.LoadByID"

	if err := service.Load(session, exp.Equal("_id", streamSourceID), streamSource); err != nil {
		return derp.Wrap(err, location, "Loading StreamSource", streamSourceID)
	}

	return nil
}

// RangeByWebhookToken returns every live StreamSource record that carries the provided webhook token
func (service *StreamSource) RangeByWebhookToken(session data.Session, token string) (iter.Seq[model.StreamSource], error) {

	const location = "service.StreamSource.RangeByWebhookToken"

	// RULE: A token this short is refused before any lookup.  An empty one would otherwise match
	// every record whose token was never set, which selects records rather than authorizing one.
	if len(token) < model.StreamSourceWebhookTokenMinLength {
		return nil, derp.BadRequest(location, "Webhook token is too short")
	}

	// The lookup is scoped to this Domain by the database it runs in, not by a filter
	criteria := exp.Equal("config."+model.StreamSourceConfigWebhookToken, token)

	result, err := service.Range(session, criteria)

	if err != nil {
		return nil, derp.Wrap(err, location, "Listing StreamSources by webhook token")
	}

	return result, nil
}

// SyncByWebhookToken queues a synchronization for every live record that carries a webhook token
func (service *StreamSource) SyncByWebhookToken(session data.Session, token string) error {

	const location = "service.StreamSource.SyncByWebhookToken"

	streamSources, err := service.RangeByWebhookToken(session, token)

	if err != nil {
		return derp.Wrap(err, location, "Listing StreamSources by webhook token")
	}

	// RULE: One task per matching record, each carrying its own signature.  That is what bounds
	// the fan-out: N records cost at most N queued tasks, however many pings arrive.
	for streamSource := range streamSources {
		service.PublishSyncTask(session, streamSource.StreamSourceID)
	}

	return nil
}

// LoadByStreamID retrieves the StreamSource record attached to a Stream
func (service *StreamSource) LoadByStreamID(session data.Session, streamID primitive.ObjectID, streamSource *model.StreamSource) error {

	const location = "service.StreamSource.LoadByStreamID"

	if err := service.Load(session, exp.Equal("streamId", streamID), streamSource); err != nil {
		return derp.Wrap(err, location, "Loading StreamSource", streamID)
	}

	return nil
}

/******************************************
 * ModelService Interface
 *
 * Implemented so that a Template can reach this
 * record through the `with-stream-source` step,
 * and edit it with the generic `edit`, `save`,
 * and `delete` steps.
 ******************************************/

// ObjectType returns the name of this service's model object. Implements the ModelService interface.
func (service *StreamSource) ObjectType() string {
	return "StreamSource"
}

// ObjectNew returns a fully initialized model.StreamSource as a data.Object.
// Implements the ModelService interface.
func (service *StreamSource) ObjectNew() data.Object {
	result := model.NewStreamSource()
	return &result
}

// ObjectID returns the primary key of a model.StreamSource. Implements the ModelService interface.
func (service *StreamSource) ObjectID(object data.Object) primitive.ObjectID {

	if streamSource, ok := object.(*model.StreamSource); ok {
		return streamSource.StreamSourceID
	}

	return primitive.NilObjectID
}

// ObjectQuery populates a slice of StreamSource records. Implements the ModelService interface.
func (service *StreamSource) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

// ObjectLoad retrieves a single StreamSource as a data.Object. Implements the ModelService interface.
func (service *StreamSource) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewStreamSource()
	err := service.Load(session, criteria, &result)
	return &result, err
}

// ObjectSave saves a StreamSource to the database. Implements the ModelService interface.
func (service *StreamSource) ObjectSave(session data.Session, object data.Object, note string) error {

	if streamSource, ok := object.(*model.StreamSource); ok {
		return service.Save(session, streamSource, note)
	}

	return derp.Internal("service.StreamSource.ObjectSave", "Invalid object type", object)
}

// ObjectDelete removes a StreamSource from the database. Implements the ModelService interface.
func (service *StreamSource) ObjectDelete(session data.Session, object data.Object, note string) error {

	if streamSource, ok := object.(*model.StreamSource); ok {
		return service.Delete(session, streamSource, note)
	}

	return derp.Internal("service.StreamSource.ObjectDelete", "Invalid object type", object)
}

// ObjectUserCan refuses every direct permission check. Implements the ModelService interface.
func (service *StreamSource) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {

	// RULE: A StreamSource is reached only through the Stream it populates, whose own action was
	// already checked.  Nothing may address this record on its own.
	return derp.Unauthorized("service.StreamSource.ObjectUserCan", "Not Authorized")
}

/******************************************
 * Adapters and Tasks
 ******************************************/

// adapterFor returns the Adapter that reads a StreamSource Method
func (service *StreamSource) adapterFor(method string) (content.Adapter, error) {

	const location = "service.StreamSource.adapterFor"

	if adapter, exists := service.adapters[method]; exists {
		return adapter, nil
	}

	// A Method the schema allows but no Adapter reads is a defect in this table, not a bad record
	return nil, derp.Internal(location, "No Adapter is registered for this Method", method)
}

// PublishSyncTask queues a BACKGROUND synchronization for a single StreamSource record
func (service *StreamSource) PublishSyncTask(session data.Session, streamSourceID primitive.ObjectID) {

	// RULE: One task per record.  A ping arriving while a sync is already queued collapses into
	// it, and loses nothing: the queued sync reads the origin when it runs, so it picks up the
	// newer commit anyway.
	postcommit.Publish(
		session,
		service.queue,
		TaskSyncStreamSource,
		service.syncTaskArguments(streamSourceID),
		queue.WithSignature("StreamSource-Sync:"+streamSourceID.Hex()),
	)
}

// PublishSyncTaskNow queues an INTERACTIVE synchronization for a single StreamSource record,
// behind a human who is watching the settings screen for the answer.
func (service *StreamSource) PublishSyncTaskNow(session data.Session, streamSourceID primitive.ObjectID) {

	// RULE: NO signature, and that is the whole reason this task is named separately.  turbine's
	// allowImmediate refuses to run ANY signed task from memory, at any priority, because
	// signature dedup needs a stored row to check against -- so a signed task waits for the
	// storage poller, which sleeps a minute when the queue is idle.
	//
	// What that costs: two quick presses queue two syncs, and one of them may overlap a webhook's
	// background sync of the same record.  A repeat is one conditional GET answering 304, so the
	// usual case is free.  The case that is NOT free is a caller that saves many records at once,
	// which now fans out immediate fetches instead of deduplicated background ones.
	postcommit.Publish(
		session,
		service.queue,
		TaskSyncStreamSourceNow,
		service.syncTaskArguments(streamSourceID),
	)
}

// syncTaskArguments builds the arguments that both synchronization tasks carry.  One handler reads
// both, so a key the two spelled differently would strand one path.
func (service *StreamSource) syncTaskArguments(streamSourceID primitive.ObjectID) mapof.Any {
	return mapof.Any{
		"hostname":       service.hostname,
		"streamSourceId": streamSourceID.Hex(),
	}
}

/******************************************
 * Status
 ******************************************/

// markChecked stamps the moment a synchronization attempt touched the origin.  The settings
// screen labels this "Last Checked", and EVERY ping moves it -- a webhook and the Sync Now button
// alike -- whether the attempt then succeeded, failed, or is about to be retried.
//
// RULE: The four failing exits of Sync never reach a save.  `consumer.WithSession` aborts the
// task's transaction, so the timestamp Sync set in memory is rolled back with it, and the
// lifecycle hook that survives reloads the record from the database.  That hook is therefore the
// only place a failed attempt can record that it happened at all.
func markChecked(streamSource *model.StreamSource) {
	streamSource.LastSynced = time.Now().Unix()
}

// SetStatusLoading marks a StreamSource record as synchronizing, and saves it
func (service *StreamSource) SetStatusLoading(session data.Session, streamSource *model.StreamSource) error {

	const location = "service.StreamSource.SetStatusLoading"

	streamSource.Status = model.StreamSourceStatusLoading
	streamSource.StatusMessage = ""
	markChecked(streamSource)

	// Status bookkeeping skips Save's validation, which a status change can never violate
	if err := service.save(session, streamSource, "Synchronizing"); err != nil {
		return derp.Wrap(err, location, "Saving StreamSource", streamSource.StreamSourceID)
	}

	return nil
}

// SetStatusSuccess marks a StreamSource record as synchronized, and saves it
func (service *StreamSource) SetStatusSuccess(session data.Session, streamSource *model.StreamSource) error {

	const location = "service.StreamSource.SetStatusSuccess"

	time.Sleep(200 * time.Millisecond)

	streamSource.Status = model.StreamSourceStatusSuccess
	streamSource.StatusMessage = ""
	markChecked(streamSource)

	if err := service.save(session, streamSource, "Synchronized"); err != nil {
		return derp.Wrap(err, location, "Saving StreamSource", streamSource.StreamSourceID)
	}

	return nil
}

// SetStatusFailure marks a StreamSource record as failed, and saves it
func (service *StreamSource) SetStatusFailure(session data.Session, streamSource *model.StreamSource, statusMessage string) error {

	const location = "service.StreamSource.SetStatusFailure"

	streamSource.Status = model.StreamSourceStatusFailure
	streamSource.StatusMessage = truncateStatusMessage(statusMessage)
	markChecked(streamSource)

	if err := service.save(session, streamSource, "Synchronization failed"); err != nil {
		return derp.Wrap(err, location, "Saving StreamSource", streamSource.StreamSourceID)
	}

	return nil
}

// SetStatusMessage records why an attempt failed, without deciding the record's Status
func (service *StreamSource) SetStatusMessage(session data.Session, streamSource *model.StreamSource, statusMessage string) error {

	const location = "service.StreamSource.SetStatusMessage"

	// RULE: Status is left alone because a retry is still queued.  FAILURE here would read as
	// broken to an author whose sync is about to succeed on its own.  The TIMESTAMP still moves:
	// an attempt that failed is still an attempt, and the message beside it describes this one.
	streamSource.StatusMessage = truncateStatusMessage(statusMessage)
	markChecked(streamSource)

	if err := service.save(session, streamSource, "Synchronization retrying"); err != nil {
		return derp.Wrap(err, location, "Saving StreamSource", streamSource.StreamSourceID)
	}

	return nil
}

/******************************************
 * Helper Functions
 ******************************************/

// validateStreamSourceURL refuses an address that cannot be read, or that carries credentials.
// Adapters apply their own stricter rules when they read it.
func validateStreamSourceURL(value string) error {

	const location = "service.validateStreamSourceURL"

	// The parser's own error quotes the whole address, so it is replaced rather than wrapped
	parsed, err := url.Parse(value)

	if err != nil {
		return derp.BadRequest(location, "URL is not valid")
	}

	// RULE: Credentials never live in the address, where they would be stored in plain text
	if parsed.User != nil {
		return derp.BadRequest(location, "URL must not include a username or password")
	}

	return nil
}

// truncateStatusMessage shortens a status message to the length its schema allows
func truncateStatusMessage(value string) string {

	const maxLength = 1024

	if len(value) <= maxLength {
		return value
	}

	// RULE: A cut can split a multi-byte character, and MongoDB stores only valid UTF-8, so the
	// broken fragment at the end is dropped
	return strings.ToValidUTF8(value[:maxLength], "")
}
