package service

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/realtime"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/EmissarySocial/emissary/tools/secretcheck"
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/derp/plugins"
	"github.com/benpate/exp"
	"github.com/benpate/form"
	"github.com/benpate/hannibal/delivery"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// streamDeleteTest holds a Stream service wired for Delete against a real MongoDB, and counts
// the signing-key loads and webhook sends that Delete starts
type streamDeleteTest struct {
	service  Stream
	session  data.Session
	keyLoads *atomic.Int32
	webhooks *atomic.Int32
	tasks    *postcommit.Tasks
}

// newStreamDeleteTest returns a Stream service wired with every service that Delete calls. A Delete is
// queued for delivery after the commit, and loading a signing key always fails and is counted.
func newStreamDeleteTest(t *testing.T) streamDeleteTest {

	t.Helper()

	_, session := newReplicaSetSession(t)

	keyLoads := &atomic.Int32{}
	webhooks := &atomic.Int32{}
	tasks := postcommit.NewTasks()

	webhookService := &Webhook{
		newSession: func(time.Duration) (data.Session, context.CancelFunc, error) {
			webhooks.Add(1)
			return nil, nil, derp.Internal("test", "Webhooks are not delivered in this test")
		},
	}

	keyService := &EncryptionKey{}
	locator := &Locator{host: "https://example.test"}

	// Outbox2's removals from the legacy outbox and the object cache are recorded, not made
	removals := &removalRecorder{}
	outbox2 := &Outbox2{locator: locator, host: "https://example.test", legacyOutbox: removals, objectCache: removals}
	legacyOutbox := &Outbox{userService: &User{keyService: keyService}, outbox2Service: outbox2}

	service := Stream{
		attachmentService:   &Attachment{},
		collectionService:   &Collection{locatorService: locator},
		domainService:       &Domain{},
		draftService:        &StreamDraft{},
		locatorService:      locator,
		outboxService:       legacyOutbox,
		outbox2Service:      outbox2,
		searchResultService: &SearchResult{},
		streamSourceService: &StreamSource{},
		webhookService:      webhookService,
		sseUpdateChannel:    make(chan realtime.Message, 100),
	}

	return streamDeleteTest{
		service:  service,
		session:  keyFailSession{Session: session, keyLoads: keyLoads, tasks: tasks},
		keyLoads: keyLoads,
		webhooks: webhooks,
		tasks:    tasks,
	}
}

// keyFailSession wraps a real session, failing every EncryptionKey load and spooling post-commit tasks
type keyFailSession struct {
	data.Session
	keyLoads *atomic.Int32
	tasks    *postcommit.Tasks
}

// Collection returns the real collection, except for EncryptionKey
func (s keyFailSession) Collection(name string) data.Collection {

	if name == "EncryptionKey" {
		return keyFailCollection{Collection: s.Session.Collection(name), keyLoads: s.keyLoads}
	}

	return s.Session.Collection(name)
}

// Context carries the post-commit spool, so queued tasks can be counted
func (s keyFailSession) Context() context.Context {
	return postcommit.WithContext(s.Session.Context(), s.tasks)
}

// keyFailCollection counts and fails every Load
type keyFailCollection struct {
	data.Collection
	keyLoads *atomic.Int32
}

// Load records the attempt and fails it
func (c keyFailCollection) Load(exp.Expression, data.Object, ...option.Option) error {
	c.keyLoads.Add(1)
	return derp.Internal("test", "Signing keys are not available in this test")
}

// saveStream stores a Stream by the given author, published when `published` is TRUE
func (test streamDeleteTest) saveStream(t *testing.T, authorID primitive.ObjectID, parentID primitive.ObjectID, published bool) model.Stream {

	t.Helper()

	stream := model.NewStream()
	stream.StreamID = primitive.NewObjectID()
	stream.ParentID = parentID
	stream.URL = "https://example.test/" + stream.StreamID.Hex()
	stream.AttributedTo.UserID = authorID

	if published {
		stream.PublishDate = time.Now().Add(-time.Hour).Unix()
	} else {
		stream.PublishDate = time.Now().Add(time.Hour).Unix()
	}

	require.NoError(t, test.session.Collection("Stream").Save(&stream, "test setup"))
	return stream
}

// saveCreate stores the outbox row that a Create of this Stream leaves in the actor's outbox
func (test streamDeleteTest) saveCreate(t *testing.T, actorID primitive.ObjectID, stream model.Stream) {
	t.Helper()
	test.saveActivity(t, actorID, stream, vocab.ActivityTypeCreate)
}

// saveActivity stores a legacy outbox row of the given activity type for this Stream
func (test streamDeleteTest) saveActivity(t *testing.T, actorID primitive.ObjectID, stream model.Stream, activityType string) {

	t.Helper()

	message := model.NewOutboxMessage()
	message.ActorType = model.FollowerTypeUser
	message.ActorID = actorID
	message.ActorURL = "https://example.test/@" + actorID.Hex()
	message.ActivityType = activityType
	message.ObjectID = stream.URL

	require.NoError(t, test.session.Collection("Outbox").Save(&message, "test setup"))
}

// saveOutbox2Activity stores an Outbox2 row of the given activity type for this object, and returns it
func (test streamDeleteTest) saveOutbox2Activity(t *testing.T, actorType string, actorID primitive.ObjectID, objectID string, activityType string) model.OutboxItem {

	t.Helper()

	item := model.NewOutboxItem()
	item.ActorType = actorType
	item.ActorID = actorID
	item.URL = "https://example.test/@" + actorID.Hex() + "/pub/outbox/" + item.ActivityID.Hex()
	item.ActivityType = activityType
	item.ObjectID = objectID

	require.NoError(t, test.session.Collection("OutboxItem").Save(&item, "test setup"))
	return item
}

// countOutbox2 returns the number of Outbox2 rows that belong to an actor
func (test streamDeleteTest) countOutbox2(t *testing.T, actorType string, actorID primitive.ObjectID) int64 {

	t.Helper()

	result, err := test.session.Collection("OutboxItem").Count(exp.Equal("actorType", actorType).AndEqual("actorId", actorID))
	require.NoError(t, err)
	return result
}

// deletesSent returns the number of Delete activities queued for delivery since it was last called
func (test streamDeleteTest) deletesSent(t *testing.T) int {

	t.Helper()

	result := 0

	for _, task := range test.tasks.Drain() {

		if task.Name != delivery.TaskFanOut {
			continue
		}

		if task.Arguments.GetMap("activity").GetString(vocab.PropertyType) == vocab.ActivityTypeDelete {
			result++
		}
	}

	return result
}

// requireCount waits for the counter to reach `want`, then confirms that it stays there
func requireCount(t *testing.T, counter *atomic.Int32, want int32, label string) {

	t.Helper()

	require.Eventually(t, func() bool { return counter.Load() >= want }, 5*time.Second, 10*time.Millisecond, label)
	time.Sleep(50 * time.Millisecond)
	require.Equal(t, want, counter.Load(), label)
}

// TestStreamDelete_FederatesADelete verifies that deleting a Stream sends a Delete as its author
// only while the author's outbox still holds the Create that federated it
func TestStreamDelete_FederatesADelete(t *testing.T) {

	// BUG-259: no delete path sent a Delete, so deleted Streams stayed on every follower's server

	t.Run("federated", func(t *testing.T) {
		test := newStreamDeleteTest(t)
		authorID := primitive.NewObjectID()
		stream := test.saveStream(t, authorID, primitive.NewObjectID(), true)
		test.saveCreate(t, authorID, stream)

		require.NoError(t, test.service.Delete(test.session, &stream, "test"))
		require.Equal(t, 1, test.deletesSent(t), "one Delete is queued as the author")
		requireCount(t, test.keyLoads, 0, "the Delete is signed by the send task, never inside the request")
	})

	t.Run("never federated", func(t *testing.T) {
		test := newStreamDeleteTest(t)
		stream := test.saveStream(t, primitive.NewObjectID(), primitive.NewObjectID(), true)

		require.NoError(t, test.service.Delete(test.session, &stream, "test"))
		require.Equal(t, 0, test.deletesSent(t), "a Stream followers never received sends no Delete")
	})

	t.Run("another actor's outbox", func(t *testing.T) {
		test := newStreamDeleteTest(t)
		stream := test.saveStream(t, primitive.NewObjectID(), primitive.NewObjectID(), true)
		test.saveCreate(t, primitive.NewObjectID(), stream)

		require.NoError(t, test.service.Delete(test.session, &stream, "test"))
		require.Equal(t, 0, test.deletesSent(t), "only the author's outbox counts")
	})

	t.Run("no author", func(t *testing.T) {
		test := newStreamDeleteTest(t)
		stream := test.saveStream(t, primitive.NilObjectID, primitive.NewObjectID(), true)

		require.NoError(t, test.service.Delete(test.session, &stream, "test"))
		require.Equal(t, 0, test.deletesSent(t), "a Stream with no author sends no Delete")
	})

	t.Run("children", func(t *testing.T) {
		test := newStreamDeleteTest(t)
		authorID := primitive.NewObjectID()
		parent := test.saveStream(t, authorID, primitive.NewObjectID(), true)
		child := test.saveStream(t, authorID, parent.StreamID, true)
		test.saveStream(t, authorID, parent.StreamID, true)
		test.saveCreate(t, authorID, parent)
		test.saveCreate(t, authorID, child)

		require.NoError(t, test.service.Delete(test.session, &parent, "test"))
		require.Equal(t, 2, test.deletesSent(t), "the parent and its one federated child each send a Delete")
	})

	t.Run("unpublished", func(t *testing.T) {

		// BUG-269: unpublishing leaves its own Delete row, which read as the Stream still being published
		test := newStreamDeleteTest(t)
		authorID := primitive.NewObjectID()
		stream := test.saveStream(t, authorID, primitive.NewObjectID(), true)
		test.saveActivity(t, authorID, stream, vocab.ActivityTypeDelete)

		require.NoError(t, test.service.Delete(test.session, &stream, "test"))
		require.Equal(t, 0, test.deletesSent(t), "an unpublished Stream was already withdrawn, so it sends no second Delete")
	})

	t.Run("updated", func(t *testing.T) {
		test := newStreamDeleteTest(t)
		authorID := primitive.NewObjectID()
		stream := test.saveStream(t, authorID, primitive.NewObjectID(), true)
		test.saveActivity(t, authorID, stream, vocab.ActivityTypeUpdate)

		require.NoError(t, test.service.Delete(test.session, &stream, "test"))
		require.Equal(t, 1, test.deletesSent(t), "an Update publishes the Stream as a Create does")
	})

	t.Run("republished", func(t *testing.T) {
		test := newStreamDeleteTest(t)
		authorID := primitive.NewObjectID()
		stream := test.saveStream(t, authorID, primitive.NewObjectID(), true)
		test.saveActivity(t, authorID, stream, vocab.ActivityTypeDelete)
		test.saveCreate(t, authorID, stream)

		require.NoError(t, test.service.Delete(test.session, &stream, "test"))
		require.Equal(t, 1, test.deletesSent(t), "a Stream published again after unpublishing sends one Delete")
	})

	t.Run("Outbox2", func(t *testing.T) {
		test := newStreamDeleteTest(t)
		authorID := primitive.NewObjectID()
		stream := test.saveStream(t, authorID, primitive.NewObjectID(), true)
		test.saveOutbox2Activity(t, model.ActorTypeUser, authorID, stream.URL, vocab.ActivityTypeCreate)

		require.NoError(t, test.service.Delete(test.session, &stream, "test"))
		require.Equal(t, 1, test.deletesSent(t), "a Create in Outbox2 counts as a Create in the legacy outbox")
	})

	t.Run("Outbox2 unpublished", func(t *testing.T) {
		test := newStreamDeleteTest(t)
		authorID := primitive.NewObjectID()
		stream := test.saveStream(t, authorID, primitive.NewObjectID(), true)
		test.saveOutbox2Activity(t, model.ActorTypeUser, authorID, stream.URL, vocab.ActivityTypeDelete)

		require.NoError(t, test.service.Delete(test.session, &stream, "test"))
		require.Equal(t, 0, test.deletesSent(t), "a Delete in Outbox2 does not count as a publication")
	})

	t.Run("another actor's Outbox2", func(t *testing.T) {
		test := newStreamDeleteTest(t)
		stream := test.saveStream(t, primitive.NewObjectID(), primitive.NewObjectID(), true)
		test.saveOutbox2Activity(t, model.ActorTypeUser, primitive.NewObjectID(), stream.URL, vocab.ActivityTypeCreate)

		require.NoError(t, test.service.Delete(test.session, &stream, "test"))
		require.Equal(t, 0, test.deletesSent(t), "only the author's Outbox2 counts")
	})
}

// TestStreamDelete_DeletesOutbox2Activities verifies that deleting a Stream removes the activities in its
// own Outbox2, and leaves another Stream's and a User's with the same ID alone
func TestStreamDelete_DeletesOutbox2Activities(t *testing.T) {

	test := newStreamDeleteTest(t)
	stream := test.saveStream(t, primitive.NilObjectID, primitive.NewObjectID(), true)
	otherStreamID := primitive.NewObjectID()

	test.saveOutbox2Activity(t, model.ActorTypeStream, stream.StreamID, "https://example.test/notes/1", vocab.ActivityTypeAnnounce)
	test.saveOutbox2Activity(t, model.ActorTypeStream, stream.StreamID, "https://example.test/notes/2", vocab.ActivityTypeAnnounce)
	test.saveOutbox2Activity(t, model.ActorTypeStream, otherStreamID, "https://example.test/notes/1", vocab.ActivityTypeAnnounce)
	test.saveOutbox2Activity(t, model.ActorTypeUser, stream.StreamID, "https://example.test/notes/3", vocab.ActivityTypeCreate)

	require.NoError(t, test.service.Delete(test.session, &stream, "test"))

	require.Zero(t, test.countOutbox2(t, model.ActorTypeStream, stream.StreamID))
	require.Equal(t, int64(1), test.countOutbox2(t, model.ActorTypeStream, otherStreamID))
	require.Equal(t, int64(1), test.countOutbox2(t, model.ActorTypeUser, stream.StreamID))
}

// streamDeleteReports is a derp.Reporter that keeps every error reported to it
type streamDeleteReports struct {
	lock   sync.Mutex
	errors []error
}

// Report records one error
func (reports *streamDeleteReports) Report(err error) {
	reports.lock.Lock()
	defer reports.lock.Unlock()
	reports.errors = append(reports.errors, err)
}

// withMessage returns the recorded errors whose outermost message matches
func (reports *streamDeleteReports) withMessage(message string) []error {

	reports.lock.Lock()
	defer reports.lock.Unlock()

	result := make([]error, 0, len(reports.errors))

	for _, err := range reports.errors {
		if derp.Message(err) == message {
			result = append(result, err)
		}
	}

	return result
}

// failingRemover is a legacy outbox that cannot remove anything
type failingRemover struct{}

// removeOutboxMessagesByObjectID always fails
func (failingRemover) removeOutboxMessagesByObjectID(data.Session, string, primitive.ObjectID, string) error {
	return derp.Internal("test", "The legacy outbox is unavailable in this test")
}

// removeOutboxMessagesByActivityURL always fails
func (failingRemover) removeOutboxMessagesByActivityURL(data.Session, string, primitive.ObjectID, string) error {
	return derp.Internal("test", "The legacy outbox is unavailable in this test")
}

// TestStreamDelete_ReportNamesTheStream verifies that a Delete which cannot be sent is reported
// with the Stream's ID, and never carries the Stream itself
func TestStreamDelete_ReportNamesTheStream(t *testing.T) {

	// BUG-270: the report attached the whole Stream, which the error log stores verbatim
	reports := &streamDeleteReports{}
	derp.SetPlugins(reports)
	t.Cleanup(func() { derp.SetPlugins(plugins.JSON{}) })

	test := newStreamDeleteTest(t)
	authorID := primitive.NewObjectID()
	stream := test.saveStream(t, authorID, primitive.NewObjectID(), true)
	stream.Label = "label-that-must-not-be-reported"
	test.saveCreate(t, authorID, stream)

	// The Delete fails before it is queued, when the legacy outbox's rows cannot be removed
	test.service.outbox2Service.legacyOutbox = failingRemover{}

	require.NoError(t, test.service.Delete(test.session, &stream, "test"))
	require.Zero(t, test.deletesSent(t), "a Delete that fails is not queued")

	federating := reports.withMessage("Federating Delete")
	require.Len(t, federating, 1)
	secretcheck.RequireAbsent(t, federating[0], stream.Label)
}

// TestStream_UnpublishOutboxStream_ErrorNamesTheStream verifies that a Delete the parent Stream
// cannot send names the Stream by URL, and never carries the Stream itself
func TestStream_UnpublishOutboxStream_ErrorNamesTheStream(t *testing.T) {

	// BUG-270: federateDelete returns this error inside the one Stream.Delete reports
	test := newStreamDeleteTest(t)

	parentTemplate := model.NewTemplate("parent", nil)
	parentTemplate.Actor.BoostChildren = true
	test.service.templateService = &Template{templates: set.Map[model.Template]{"parent": parentTemplate}}

	// The Delete fails before it is queued, when the legacy outbox's rows cannot be removed
	test.service.outbox2Service.legacyOutbox = failingRemover{}

	stream := test.saveStream(t, primitive.NewObjectID(), primitive.NewObjectID(), true)
	stream.ParentTemplateID = "parent"
	stream.Label = "label-that-must-not-be-reported"

	err := test.service.unpublish_outbox_stream(test.session, &stream)
	require.Error(t, err)
	require.Zero(t, test.deletesSent(t), "a Delete that fails is not queued")
	secretcheck.RequireAbsent(t, err, stream.Label)
}

// TestStreamDelete_UndoesPublication verifies that deleting a published Stream sends the
// publish:undo webhook and queues a syndication delete, which the delete itself used to hide
func TestStreamDelete_UndoesPublication(t *testing.T) {

	// BUG-241: the published check ran after the delete marked the Stream deleted, so it never passed

	countSyndication := func(test streamDeleteTest) int {
		result := 0
		for _, task := range test.tasks.Drain() {
			if task.Name == "syndication.delete" {
				result++
			}
		}
		return result
	}

	withTarget := func(test streamDeleteTest) {
		domain := model.NewWritableDomain()
		domain.Syndication = sliceof.Object[form.LookupCode]{{Value: "bluesky", Href: "https://example.test/syndicate"}}
		test.service.domainService.cache.Store(&domain)
	}

	t.Run("published", func(t *testing.T) {
		test := newStreamDeleteTest(t)
		withTarget(test)
		stream := test.saveStream(t, primitive.NewObjectID(), primitive.NewObjectID(), true)
		stream.Syndication.Values = []string{"bluesky"}

		require.NoError(t, test.service.Delete(test.session, &stream, "test"))
		requireCount(t, test.webhooks, 2, "stream:delete and stream:publish:undo")
		require.Equal(t, 1, countSyndication(test))
	})

	t.Run("unpublished", func(t *testing.T) {
		test := newStreamDeleteTest(t)
		withTarget(test)
		stream := test.saveStream(t, primitive.NewObjectID(), primitive.NewObjectID(), false)
		stream.Syndication.Values = []string{"bluesky"}

		require.NoError(t, test.service.Delete(test.session, &stream, "test"))
		requireCount(t, test.webhooks, 1, "stream:delete only")
		require.Zero(t, countSyndication(test))
	})
}
