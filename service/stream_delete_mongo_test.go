package service

import (
	"context"
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
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/delivery"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// streamDeleteTest holds a Stream service wired for Delete against a real MongoDB, and counts
// the signing-key loads that Delete starts
type streamDeleteTest struct {
	service  Stream
	session  data.Session
	keyLoads *atomic.Int32
	tasks    *postcommit.Tasks
}

// newStreamDeleteTest returns a Stream service wired with every service that Delete calls. A Delete is
// queued for delivery after the commit, and loading a signing key always fails and is counted.
func newStreamDeleteTest(t *testing.T) streamDeleteTest {

	t.Helper()

	_, session := newReplicaSetSession(t)

	keyLoads := &atomic.Int32{}
	tasks := postcommit.NewTasks()

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
		sseUpdateChannel:    make(chan realtime.Message, 100),
	}

	return streamDeleteTest{
		service:  service,
		session:  keyFailSession{Session: session, keyLoads: keyLoads, tasks: tasks},
		keyLoads: keyLoads,
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

// saveStream stores a Stream by the given author, with a past publish date when `published` is
// TRUE.  The date decides nothing: only saveSentStream puts a Stream in the outbox.
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

// outboxTasks returns the StreamIDs of the outbox tasks queued since it was last called, and the
// number of Deletes queued for delivery beside them
func (test streamDeleteTest) outboxTasks(t *testing.T) (streamIDs []string, deletes int) {

	t.Helper()

	streamIDs = make([]string, 0)

	for _, task := range test.tasks.Drain() {

		switch task.Name {

		case StreamOutboxTaskName:
			streamIDs = append(streamIDs, task.Arguments.GetString("streamId"))

		case delivery.TaskFanOut:
			if task.Arguments.GetMap("activity").GetString(vocab.PropertyType) == vocab.ActivityTypeDelete {
				deletes++
			}
		}
	}

	return streamIDs, deletes
}

// saveSentStream stores a Stream by the given author that followers hold a copy of
func (test streamDeleteTest) saveSentStream(t *testing.T, authorID primitive.ObjectID, parentID primitive.ObjectID) model.Stream {

	t.Helper()

	stream := test.saveStream(t, authorID, parentID, true)
	stream.IsOutbox.Set(true)
	stream.Fingerprint.Set("what-followers-hold")
	stream.MarkSaved()

	require.NoError(t, test.session.Collection("Stream").Save(&stream, "test setup"))
	return stream
}

// TestStreamDelete_QueuesTheOutboxTask verifies that deleting a Stream which followers hold queues the
// outbox task, on every delete path, and sends nothing inside the request.
// TestStream_SyncOutbox_Delete covers what the task then sends.
func TestStreamDelete_QueuesTheOutboxTask(t *testing.T) {

	// BUG-259: no delete path sent a Delete, so deleted Streams stayed on every follower's server

	t.Run("followers hold it", func(t *testing.T) {
		test := newStreamDeleteTest(t)
		stream := test.saveSentStream(t, primitive.NewObjectID(), primitive.NewObjectID())

		require.NoError(t, test.service.Delete(test.session, &stream, "test"))

		streamIDs, deletes := test.outboxTasks(t)
		require.Equal(t, []string{stream.StreamID.Hex()}, streamIDs)
		require.Zero(t, deletes, "the Delete is sent by the task, never inside the request")
		requireCount(t, test.keyLoads, 0, "the Delete is signed by the send task, never inside the request")
	})

	t.Run("followers hold nothing", func(t *testing.T) {

		// BUG-269: an unpublished Stream was already withdrawn, so it sends no second Delete
		test := newStreamDeleteTest(t)
		stream := test.saveStream(t, primitive.NewObjectID(), primitive.NewObjectID(), true)

		require.NoError(t, test.service.Delete(test.session, &stream, "test"))

		streamIDs, deletes := test.outboxTasks(t)
		require.Empty(t, streamIDs)
		require.Zero(t, deletes)
	})

	t.Run("children", func(t *testing.T) {
		test := newStreamDeleteTest(t)
		authorID := primitive.NewObjectID()
		parent := test.saveSentStream(t, authorID, primitive.NewObjectID())
		child := test.saveSentStream(t, authorID, parent.StreamID)
		test.saveStream(t, authorID, parent.StreamID, true)

		require.NoError(t, test.service.Delete(test.session, &parent, "test"))

		streamIDs, deletes := test.outboxTasks(t)
		require.ElementsMatch(t, []string{parent.StreamID.Hex(), child.StreamID.Hex()}, streamIDs, "the parent and its one sent child")
		require.Zero(t, deletes)
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

// TestStream_SyncOutbox_ErrorNamesTheStream verifies that a Delete which cannot be sent fails the
// outbox task with the Stream's ID, and never carries the Stream itself
func TestStream_SyncOutbox_ErrorNamesTheStream(t *testing.T) {

	// BUG-270: the report attached the whole Stream, which the error log stores verbatim
	test := newStreamDeleteTest(t)
	test.service.userService = &User{}

	author := model.NewUser()
	require.NoError(t, test.session.Collection("User").Save(&author, "test setup"))

	stream := test.saveSentStream(t, author.UserID, primitive.NewObjectID())
	stream.Label = "label-that-must-not-be-reported"
	require.NoError(t, test.session.Collection("Stream").Save(&stream, "test setup"))
	require.NoError(t, test.service.Delete(test.session, &stream, "test"))

	// The Delete fails before it is queued, when the legacy outbox's rows cannot be removed
	test.service.outbox2Service.legacyOutbox = failingRemover{}

	err := test.service.SyncOutbox(test.session, stream.StreamID, OutboxChanges{})
	require.Error(t, err)
	secretcheck.RequireAbsent(t, err, stream.Label)

	_, deletes := test.outboxTasks(t)
	require.Zero(t, deletes, "a Delete that fails is not queued")
}

// TestStream_UnpublishOutboxStream_ErrorNamesTheStream verifies that a Delete the parent Stream
// cannot send names the Stream by URL, and never carries the Stream itself
func TestStream_UnpublishOutboxStream_ErrorNamesTheStream(t *testing.T) {

	// BUG-270: the outbox task returns this error inside its own
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

// TestStream_UnpublishOutboxStream_RemovesTheAnnounce verifies that unpublishing a child removes the
// parent Stream's Announce of it, and sends the parent's followers a Delete
func TestStream_UnpublishOutboxStream_RemovesTheAnnounce(t *testing.T) {

	// OUTBOX2-OMNIBUS §7.8 O15: the Announce names the child's Create, so a lookup by the child missed it
	test := newStreamDeleteTest(t)

	parentTemplate := model.NewTemplate("parent", nil)
	parentTemplate.Actor.BoostChildren = true
	test.service.templateService = &Template{templates: set.Map[model.Template]{"parent": parentTemplate}}

	authorID := primitive.NewObjectID()
	parentID := primitive.NewObjectID()
	stream := test.saveStream(t, authorID, parentID, true)
	stream.ParentTemplateID = "parent"

	// announce stores the parent's Announce of a Create of the object, as publish_outbox_stream writes it
	announce := func(objectURL string) model.OutboxItem {

		createURL := "https://example.test/@" + authorID.Hex() + "/pub/outbox/" + primitive.NewObjectID().Hex()
		item := test.saveOutbox2Activity(t, model.ActorTypeStream, parentID, createURL, vocab.ActivityTypeAnnounce)
		item.Activity = mapof.Any{
			vocab.PropertyType:  vocab.ActivityTypeAnnounce,
			vocab.PropertyActor: "https://example.test/" + parentID.Hex(),
			vocab.PropertyObject: mapof.Any{
				vocab.PropertyID:     createURL,
				vocab.PropertyType:   vocab.ActivityTypeCreate,
				vocab.PropertyObject: mapof.Any{vocab.PropertyID: objectURL, vocab.PropertyType: vocab.ObjectTypeArticle},
			},
		}

		require.NoError(t, test.session.Collection("OutboxItem").Save(&item, "test setup"))
		return item
	}

	announced := announce(stream.URL)
	sibling := announce("https://example.test/" + primitive.NewObjectID().Hex())

	require.NoError(t, test.service.unpublish_outbox_stream(test.session, &stream))
	require.Equal(t, 1, test.deletesSent(t), "the parent's followers are told the child is gone")

	// The child's Announce is gone, its sibling's stays, and the parent's Delete takes its place
	remaining := model.NewOutboxItem()
	err := test.service.outbox2Service.LoadByID(test.session, model.ActorTypeStream, parentID, announced.ActivityID, &remaining)
	require.True(t, derp.IsNotFound(err), "the unpublished child's Announce is removed")
	require.NoError(t, test.service.outbox2Service.LoadByID(test.session, model.ActorTypeStream, parentID, sibling.ActivityID, &remaining))
	require.Equal(t, int64(2), test.countOutbox2(t, model.ActorTypeStream, parentID))
}
