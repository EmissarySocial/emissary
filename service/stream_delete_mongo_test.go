package service

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/realtime"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/form"
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

// newStreamDeleteTest returns a Stream service wired with every service that Delete calls.
// Loading a signing key always fails, so a Delete that is sent stops there and is counted.
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

	service := Stream{
		attachmentService:   &Attachment{},
		collectionService:   &Collection{locatorService: &Locator{host: "https://example.test"}},
		domainService:       &Domain{},
		draftService:        &StreamDraft{},
		outboxService:       &Outbox{userService: &User{keyService: keyService}},
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

	message := model.NewOutboxMessage()
	message.ActorType = model.FollowerTypeUser
	message.ActorID = actorID
	message.ActorURL = "https://example.test/@" + actorID.Hex()
	message.ActivityType = "Create"
	message.ObjectID = stream.URL

	require.NoError(t, test.session.Collection("Outbox").Save(&message, "test setup"))
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
		requireCount(t, test.keyLoads, 1, "one Delete is signed by the author")
	})

	t.Run("never federated", func(t *testing.T) {
		test := newStreamDeleteTest(t)
		stream := test.saveStream(t, primitive.NewObjectID(), primitive.NewObjectID(), true)

		require.NoError(t, test.service.Delete(test.session, &stream, "test"))
		requireCount(t, test.keyLoads, 0, "a Stream followers never received sends no Delete")
	})

	t.Run("another actor's outbox", func(t *testing.T) {
		test := newStreamDeleteTest(t)
		stream := test.saveStream(t, primitive.NewObjectID(), primitive.NewObjectID(), true)
		test.saveCreate(t, primitive.NewObjectID(), stream)

		require.NoError(t, test.service.Delete(test.session, &stream, "test"))
		requireCount(t, test.keyLoads, 0, "only the author's outbox counts")
	})

	t.Run("no author", func(t *testing.T) {
		test := newStreamDeleteTest(t)
		stream := test.saveStream(t, primitive.NilObjectID, primitive.NewObjectID(), true)

		require.NoError(t, test.service.Delete(test.session, &stream, "test"))
		requireCount(t, test.keyLoads, 0, "a Stream with no author sends no Delete")
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
		requireCount(t, test.keyLoads, 2, "the parent and its one federated child each send a Delete")
	})
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
