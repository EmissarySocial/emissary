package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/exp"
	"github.com/benpate/form"
	"github.com/benpate/hannibal/clients"
	"github.com/benpate/hannibal/delivery"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// These tests run the outbox task against a real MongoDB, because the task reads the Stream that a
// save stored, and writes the fingerprint with an update of its own.

// outboxSyncTest holds a Stream service wired to save, publish, and delete Streams, and the spool
// that collects the tasks those calls queue
type outboxSyncTest struct {
	service *Stream
	session data.Session
	tasks   *postcommit.Tasks
	author  model.User
}

// newOutboxSyncTest returns a Stream service on a throwaway database, with outboxTestStates on its
// test collection and one stored User to write as
func newOutboxSyncTest(t *testing.T) outboxSyncTest {
	t.Helper()
	return newOutboxSyncTestOn(t, newStreamTokenDatabase(t))
}

// newOutboxSyncTestOn returns an outboxSyncTest on the provided database, as one more server
// that shares nothing with the others but that database
func newOutboxSyncTestOn(t *testing.T, database *mongo.Database) outboxSyncTest {

	t.Helper()

	const host = "https://example.test"

	server := mongodb.NewServer(database)
	tasks := postcommit.NewTasks()

	session, err := server.Session(postcommit.WithContext(t.Context(), tasks))
	require.NoError(t, err)

	locator := &Locator{host: host}

	// The app's client caches published objects in this database, and never reaches the network
	ruleService := NewRule()
	activityService := NewActivityStream()
	activityService.hostname = "localhost"
	activityService.carpool = clients.NewCarpool()
	activityService.ruleService = &ruleService
	activityService.getCommonDatabase = func() data.Server { return server }
	activityService.newSession = func(time.Duration) (data.Session, context.CancelFunc, error) {
		return session, func() {}, nil
	}

	// Outbox2's removals from the legacy outbox and the object cache are recorded, not made
	removals := &removalRecorder{}
	outbox2 := &Outbox2{locator: locator, host: host, legacyOutbox: removals, objectCache: removals}

	service := newStreamTokenService()
	setOutboxTestStates(service, outboxTestStates())
	service.activityService = &activityService
	service.attachmentService = &Attachment{}
	service.collectionService = &Collection{locatorService: locator}
	service.domainService = &Domain{}
	service.draftService = &StreamDraft{}
	service.locatorService = locator
	service.outboxService = &Outbox{userService: &User{keyService: &EncryptionKey{}}, outbox2Service: outbox2}
	service.outbox2Service = outbox2
	service.searchResultService = &SearchResult{}
	service.streamSourceService = &StreamSource{}

	author := model.NewUser()
	author.ProfileURL = locator.UserURL(author.UserID)
	require.NoError(t, session.Collection("User").Save(&author, "test setup"))

	return outboxSyncTest{service: service, session: session, tasks: tasks, author: author}
}

// newStream returns an unsaved Stream by the test's author, in the provided State
func (test outboxSyncTest) newStream(stateID string) model.Stream {
	stream := newOutboxStream(stateID)
	stream.Label = "Judgment Day"
	stream.SetAttributedTo(test.author.PersonLink())
	return stream
}

// sync runs every outbox task that is queued, and returns the types of the activities they sent,
// in order.  Activities that were queued before the first outbox task count too.
func (test outboxSyncTest) sync(t *testing.T) []string {

	t.Helper()

	result := make([]string, 0)

	for _, task := range test.tasks.Drain() {

		switch task.Name {

		case StreamOutboxTaskName:
			streamID, err := primitive.ObjectIDFromHex(task.Arguments.GetString("streamId"))
			require.NoError(t, err)
			changes := OutboxChanges{
				SyndicationAdded:   task.Arguments.GetSliceOfString("syndicationAdded"),
				SyndicationDeleted: task.Arguments.GetSliceOfString("syndicationDeleted"),
				Republish:          task.Arguments.GetBool("republish"),
			}

			require.NoError(t, test.service.SyncOutbox(test.session, streamID, changes))
			result = append(result, test.sync(t)...)

		case delivery.TaskFanOut:
			result = append(result, task.Arguments.GetMap("activity").GetString(vocab.PropertyType))

		// A message to a syndication target, named by its type and its endpoint
		case "syndication.create", "syndication.update", "syndication.delete":
			result = append(result, task.Name+" "+task.Arguments.GetString("endpoint"))
		}
	}

	return result
}

// fingerprint returns the fingerprint stored for a Stream, deleted or not
func (test outboxSyncTest) fingerprint(t *testing.T, streamID primitive.ObjectID) string {

	t.Helper()

	stored := model.NewStream()
	require.NoError(t, test.session.Collection("Stream").Load(exp.Equal("_id", streamID), &stored))
	return stored.Fingerprint.Value()
}

// TestStream_SyncOutbox_FollowsTheStream walks one Stream into the outbox, through an edit, and out
// again, and confirms that followers are sent each change exactly once
func TestStream_SyncOutbox_FollowsTheStream(t *testing.T) {

	test := newOutboxSyncTest(t)
	stream := test.newStream("default")

	// A Stream outside the outbox sends nothing
	require.NoError(t, test.service.Save(test.session, &stream, "created"))
	require.Empty(t, test.sync(t))
	require.Empty(t, test.fingerprint(t, stream.StreamID))

	// Entering the outbox sends a Create
	stream.StateID.Set("published")
	require.NoError(t, test.service.Save(test.session, &stream, "published"))
	require.Equal(t, []string{vocab.ActivityTypeCreate}, test.sync(t))

	created := test.fingerprint(t, stream.StreamID)
	require.Len(t, created, 64)

	// Running the task again sends nothing
	require.NoError(t, test.service.SyncOutbox(test.session, stream.StreamID, OutboxChanges{}))
	require.Empty(t, test.sync(t))

	// A save that changes nothing followers see sends nothing
	loaded := storedStream(t, test.service, test.session, stream.StreamID)
	require.NoError(t, test.service.Save(test.session, &loaded, "unchanged"))
	require.Empty(t, test.sync(t))
	require.Equal(t, created, test.fingerprint(t, stream.StreamID))

	// An edit sends an Update
	loaded.Label = "No Fate"
	require.NoError(t, test.service.Save(test.session, &loaded, "edited"))
	require.Equal(t, []string{vocab.ActivityTypeUpdate}, test.sync(t))

	updated := test.fingerprint(t, stream.StreamID)
	require.Len(t, updated, 64)
	require.NotEqual(t, created, updated)

	// Leaving the outbox sends a Delete
	loaded = storedStream(t, test.service, test.session, stream.StreamID)
	loaded.StateID.Set("hidden")
	require.NoError(t, test.service.Save(test.session, &loaded, "withdrawn"))
	require.Equal(t, []string{vocab.ActivityTypeDelete}, test.sync(t))
	require.Empty(t, test.fingerprint(t, stream.StreamID))

	// Deleting it afterwards sends no second Delete (BUG-269)
	loaded = storedStream(t, test.service, test.session, stream.StreamID)
	require.NoError(t, test.service.Delete(test.session, &loaded, "deleted"))
	require.Empty(t, test.sync(t))
}

// TestStream_SyncOutbox_Delete confirms that deleting a Stream that followers hold sends one Delete,
// whichever path deleted it (BUG-259)
func TestStream_SyncOutbox_Delete(t *testing.T) {

	test := newOutboxSyncTest(t)

	stream := test.newStream("published")
	require.NoError(t, test.service.Save(test.session, &stream, "published"))
	require.Equal(t, []string{vocab.ActivityTypeCreate}, test.sync(t))

	loaded := storedStream(t, test.service, test.session, stream.StreamID)
	require.NoError(t, test.service.Delete(test.session, &loaded, "deleted"))
	require.Equal(t, []string{vocab.ActivityTypeDelete}, test.sync(t))
	require.Empty(t, test.fingerprint(t, stream.StreamID))

	// Running the task again sends nothing
	require.NoError(t, test.service.SyncOutbox(test.session, stream.StreamID, OutboxChanges{}))
	require.Empty(t, test.sync(t))
}

// TestStream_SyncOutbox_Children confirms that deleting a parent sends a Delete for each child that
// followers hold, and for no other
func TestStream_SyncOutbox_Children(t *testing.T) {

	test := newOutboxSyncTest(t)

	parent := test.newStream("published")
	require.NoError(t, test.service.Save(test.session, &parent, "parent"))

	sent := test.newStream("published")
	sent.ParentID = parent.StreamID
	require.NoError(t, test.service.Save(test.session, &sent, "sent child"))

	draft := test.newStream("default")
	draft.ParentID = parent.StreamID
	require.NoError(t, test.service.Save(test.session, &draft, "draft child"))

	require.Equal(t, []string{vocab.ActivityTypeCreate, vocab.ActivityTypeCreate}, test.sync(t))

	loaded := storedStream(t, test.service, test.session, parent.StreamID)
	require.NoError(t, test.service.Delete(test.session, &loaded, "deleted"))
	require.Equal(t, []string{vocab.ActivityTypeDelete, vocab.ActivityTypeDelete}, test.sync(t))
}

// TestStream_SyncOutbox_SendsNothing confirms the cases where the task has nobody to send as
func TestStream_SyncOutbox_SendsNothing(t *testing.T) {

	t.Run("a Stream that does not exist", func(t *testing.T) {
		test := newOutboxSyncTest(t)
		require.NoError(t, test.service.SyncOutbox(test.session, primitive.NewObjectID(), OutboxChanges{}))
		require.Empty(t, test.sync(t))
	})

	t.Run("no author", func(t *testing.T) {
		test := newOutboxSyncTest(t)
		stream := newOutboxStream("published")
		require.NoError(t, test.service.Save(test.session, &stream, "published"))
		require.Empty(t, test.sync(t))
		require.Empty(t, test.fingerprint(t, stream.StreamID))
	})

	t.Run("an author who no longer exists", func(t *testing.T) {

		// Deleting a User deletes their Streams, and must not send a Delete for each one
		test := newOutboxSyncTest(t)
		stream := test.newStream("published")
		require.NoError(t, test.service.Save(test.session, &stream, "published"))
		require.Equal(t, []string{vocab.ActivityTypeCreate}, test.sync(t))

		require.NoError(t, test.session.Collection("User").HardDelete(exp.Equal("_id", test.author.UserID)))

		loaded := storedStream(t, test.service, test.session, stream.StreamID)
		require.NoError(t, test.service.Delete(test.session, &loaded, "deleted"))
		require.Empty(t, test.sync(t))
	})
}

// TestStream_SyncOutbox_IncludesTheSocialMapping confirms that the fingerprint covers what the
// Template's social mapping writes, so that a change which only the mapping shows is still sent
func TestStream_SyncOutbox_IncludesTheSocialMapping(t *testing.T) {

	test := newOutboxSyncTest(t)

	stream := test.newStream("published")
	require.NoError(t, test.service.Save(test.session, &stream, "published"))
	require.Equal(t, []string{vocab.ActivityTypeCreate}, test.sync(t))

	// The Template now maps its Streams to another type
	template := test.service.templateService.templates["test-collection"]
	template.SocialRole = vocab.ObjectTypeArticle
	test.service.templateService.templates["test-collection"] = template

	require.NoError(t, test.service.SyncOutbox(test.session, stream.StreamID, OutboxChanges{}))
	require.Equal(t, []string{vocab.ActivityTypeUpdate}, test.sync(t))

	// The new mapping is now what followers hold
	require.NoError(t, test.service.SyncOutbox(test.session, stream.StreamID, OutboxChanges{}))
	require.Empty(t, test.sync(t))
}

// TestStream_SyncOutbox_TemplateMissing confirms that a Stream whose Template cannot be loaded
// fails the task, and sends no object built without its mapping
func TestStream_SyncOutbox_TemplateMissing(t *testing.T) {

	test := newOutboxSyncTest(t)

	stream := test.newStream("published")
	require.NoError(t, test.service.Save(test.session, &stream, "published"))
	test.tasks.Drain()

	delete(test.service.templateService.templates, "test-collection")

	require.Error(t, test.service.SyncOutbox(test.session, stream.StreamID, OutboxChanges{}))
	require.Empty(t, test.sync(t))
	require.Empty(t, test.fingerprint(t, stream.StreamID))
}

// TestStream_Publish_SendsThroughTheTask confirms that Publish and UnPublish send nothing
// themselves, so that followers hear of each change once
func TestStream_Publish_SendsThroughTheTask(t *testing.T) {

	test := newOutboxSyncTest(t)

	stream := newOutboxStream("default")
	stream.Label = "Judgment Day"
	require.NoError(t, test.service.Save(test.session, &stream, "created"))

	// Publish makes the User the author, and the task sends the one Create
	require.NoError(t, test.service.Publish(test.session, &test.author, &stream, "published", false))
	require.Equal(t, test.author.UserID, stream.AttributedTo.UserID)
	require.Equal(t, []string{vocab.ActivityTypeCreate}, test.sync(t))

	// UnPublish moves the Stream out, and the task sends the one Delete.  The next request
	// loads the Stream again, which is how it learns the fingerprint that the task recorded.
	stream = storedStream(t, test.service, test.session, stream.StreamID)
	require.NoError(t, test.service.UnPublish(test.session, &stream, "default"))
	require.Equal(t, []string{vocab.ActivityTypeDelete}, test.sync(t))
}

// TestStream_SyncOutbox_TwoServersSendOnce runs the outbox task for one Stream on several servers
// at once, and confirms that the database lets only one of them send
func TestStream_SyncOutbox_TwoServersSendOnce(t *testing.T) {

	const servers = 8

	// race runs the task on every server at the same moment, and returns everything they sent
	race := func(t *testing.T, tests []outboxSyncTest, streamID primitive.ObjectID) []string {

		t.Helper()

		start := make(chan struct{})
		errs := make([]error, len(tests))
		group := sync.WaitGroup{}

		for index := range tests {
			group.Go(func() {
				<-start
				errs[index] = tests[index].service.SyncOutbox(tests[index].session, streamID, OutboxChanges{})
			})
		}

		close(start)
		group.Wait()

		sent := make([]string, 0)

		for index := range tests {
			require.NoError(t, errs[index])
			sent = append(sent, tests[index].sync(t)...)
		}

		return sent
	}

	database := newStreamTokenDatabase(t)
	tests := make([]outboxSyncTest, servers)

	for index := range tests {
		tests[index] = newOutboxSyncTestOn(t, database)
	}

	first := tests[0]

	// A Create
	stream := first.newStream("published")
	require.NoError(t, first.service.Save(first.session, &stream, "published"))
	first.tasks.Drain()
	require.Equal(t, []string{vocab.ActivityTypeCreate}, race(t, tests, stream.StreamID))

	// An Update
	loaded := storedStream(t, first.service, first.session, stream.StreamID)
	loaded.Label = "No Fate"
	require.NoError(t, first.service.Save(first.session, &loaded, "edited"))
	first.tasks.Drain()
	require.Equal(t, []string{vocab.ActivityTypeUpdate}, race(t, tests, stream.StreamID))

	// A Delete
	loaded = storedStream(t, first.service, first.session, stream.StreamID)
	require.NoError(t, first.service.Delete(first.session, &loaded, "deleted"))
	first.tasks.Drain()
	require.Equal(t, []string{vocab.ActivityTypeDelete}, race(t, tests, stream.StreamID))
}

// withSyndicationTargets gives the test's Domain the named syndication targets
func (test outboxSyncTest) withSyndicationTargets(names ...string) {

	domain := model.NewWritableDomain()

	for _, name := range names {
		domain.Syndication = append(domain.Syndication, form.LookupCode{Value: name, Href: name})
	}

	test.service.domainService.cache.Store(&domain)
}

// TestStream_SyncOutbox_Syndication walks one Stream through the outbox, and confirms that its
// syndication targets are told what its followers are told
func TestStream_SyncOutbox_Syndication(t *testing.T) {

	test := newOutboxSyncTest(t)
	test.withSyndicationTargets("bandwagon", "spotify", "tidal")

	// A Stream outside the outbox syndicates nothing
	stream := test.newStream("default")
	stream.Syndication.Set([]string{"bandwagon", "spotify"})
	require.NoError(t, test.service.Save(test.session, &stream, "created"))
	require.Empty(t, test.sync(t))

	// Entering the outbox sends each target a Create
	stream.StateID.Set("published")
	require.NoError(t, test.service.Save(test.session, &stream, "published"))
	require.ElementsMatch(t, []string{vocab.ActivityTypeCreate, "syndication.create bandwagon", "syndication.create spotify"}, test.sync(t))

	// An edit sends each target an Update, which only the publish action used to do
	loaded := storedStream(t, test.service, test.session, stream.StreamID)
	loaded.Label = "No Fate"
	require.NoError(t, test.service.Save(test.session, &loaded, "edited"))
	require.ElementsMatch(t, []string{vocab.ActivityTypeUpdate, "syndication.update bandwagon", "syndication.update spotify"}, test.sync(t))

	// Changing the targets alone tells only the targets that changed, and no follower
	loaded = storedStream(t, test.service, test.session, stream.StreamID)
	loaded.Syndication.Set([]string{"bandwagon", "tidal"})
	require.NoError(t, test.service.Save(test.session, &loaded, "retargeted"))
	require.ElementsMatch(t, []string{"syndication.create tidal", "syndication.delete spotify"}, test.sync(t))

	// A second save of the same copy does not report the same changes again
	require.NoError(t, test.service.Save(test.session, &loaded, "saved again"))
	require.Empty(t, test.sync(t))

	// Publishing again with `republish` sends every target an Update, and no follower
	loaded = storedStream(t, test.service, test.session, stream.StreamID)
	require.NoError(t, test.service.Publish(test.session, &test.author, &loaded, "published", true))
	require.ElementsMatch(t, []string{"syndication.update bandwagon", "syndication.update tidal"}, test.sync(t))

	// Publishing again without it sends nothing
	loaded = storedStream(t, test.service, test.session, stream.StreamID)
	require.NoError(t, test.service.Publish(test.session, &test.author, &loaded, "published", false))
	require.Empty(t, test.sync(t))

	// Leaving the outbox sends each target a Delete
	loaded = storedStream(t, test.service, test.session, stream.StreamID)
	require.NoError(t, test.service.UnPublish(test.session, &loaded, "hidden"))
	require.ElementsMatch(t, []string{vocab.ActivityTypeDelete, "syndication.delete bandwagon", "syndication.delete tidal"}, test.sync(t))

	// ...and only once
	loaded = storedStream(t, test.service, test.session, stream.StreamID)
	require.NoError(t, test.service.Delete(test.session, &loaded, "deleted"))
	require.Empty(t, test.sync(t))
}

// TestStream_SyncOutbox_SyndicationDelete confirms that deleting a Stream which followers hold
// sends each syndication target a Delete, which the delete itself used to hide (BUG-241)
func TestStream_SyncOutbox_SyndicationDelete(t *testing.T) {

	test := newOutboxSyncTest(t)
	test.withSyndicationTargets("bandwagon")

	stream := test.newStream("published")
	stream.Syndication.Set([]string{"bandwagon"})
	require.NoError(t, test.service.Save(test.session, &stream, "published"))
	require.ElementsMatch(t, []string{vocab.ActivityTypeCreate, "syndication.create bandwagon"}, test.sync(t))

	loaded := storedStream(t, test.service, test.session, stream.StreamID)
	require.NoError(t, test.service.Delete(test.session, &loaded, "deleted"))
	require.ElementsMatch(t, []string{vocab.ActivityTypeDelete, "syndication.delete bandwagon"}, test.sync(t))
}

// newPublicStream returns an unsaved Stream by the test's author that anyone may view in every State
func (test outboxSyncTest) newPublicStream(stateID string) model.Stream {

	// Let the "viewer" role see the test collection in each State
	template := test.service.templateService.templates["test-collection"]
	action := template.Actions["view"]
	action.AccessList = mapof.Object[sliceof.String]{}

	for name := range template.States {
		action.AccessList[name] = sliceof.String{"author", "viewer"}
	}

	template.Actions["view"] = action
	test.service.templateService.templates["test-collection"] = template

	stream := test.newStream(stateID)
	stream.Groups["viewer"] = id.Slice{model.MagicGroupIDAnonymous}
	return stream
}

// TestStream_Save_JoinsItsConversation confirms that the save which moves a Stream into the outbox
// gives it a context and records it in a local parent's Replies, whichever caller made that save
func TestStream_Save_JoinsItsConversation(t *testing.T) {

	replyCount := func(t *testing.T, test outboxSyncTest, parent model.Stream) int {
		t.Helper()
		return storedStream(t, test.service, test.session, parent.StreamID).ReplyCount
	}

	t.Run("a draft has no context", func(t *testing.T) {
		test := newOutboxSyncTest(t)
		stream := test.newPublicStream("default")
		require.NoError(t, test.service.Save(test.session, &stream, "created"))
		require.Empty(t, stream.Context)
	})

	t.Run("entering the outbox", func(t *testing.T) {
		test := newOutboxSyncTest(t)

		parent := test.newPublicStream("published")
		require.NoError(t, test.service.Save(test.session, &parent, "parent"))
		require.NotEmpty(t, parent.Context, "an original post starts a context of its own")
		require.Equal(t, parent.Context, storedStream(t, test.service, test.session, parent.StreamID).Context)

		// Sending the parent puts its object in the cache, where the reply's save reads its context
		require.Equal(t, []string{vocab.ActivityTypeCreate}, test.sync(t))

		// A reply written as a draft is not recorded until it is sent
		reply := test.newPublicStream("default")
		reply.InReplyTo = parent.ActivityPubURL()
		require.NoError(t, test.service.Save(test.session, &reply, "draft reply"))
		require.Zero(t, replyCount(t, test, parent))

		reply.StateID.Set("published")
		require.NoError(t, test.service.Save(test.session, &reply, "sent reply"))
		require.Equal(t, 1, replyCount(t, test, parent))

		// The reply joins its parent's context, and the object that followers are sent carries it
		require.Equal(t, parent.Context, reply.Context)
		require.Contains(t, test.sync(t), vocab.ActivityTypeCreate)
		object := test.service.JSONLD(test.session, &reply)
		require.Equal(t, parent.Context, object.GetString(vocab.PropertyContext))
	})

	t.Run("a later save does not record the reply again", func(t *testing.T) {
		test := newOutboxSyncTest(t)

		parent := test.newPublicStream("published")
		require.NoError(t, test.service.Save(test.session, &parent, "parent"))
		require.Equal(t, []string{vocab.ActivityTypeCreate}, test.sync(t))

		reply := test.newPublicStream("published")
		reply.InReplyTo = parent.ActivityPubURL()
		require.NoError(t, test.service.Save(test.session, &reply, "sent reply"))

		loaded := storedStream(t, test.service, test.session, reply.StreamID)
		loaded.Label = "No Fate"
		require.NoError(t, test.service.Save(test.session, &loaded, "edited"))
		require.Equal(t, 1, replyCount(t, test, parent))
	})
}
