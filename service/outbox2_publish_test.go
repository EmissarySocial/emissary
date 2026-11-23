package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/delivery"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// These tests run Delete and Undo against a real MongoDB, because they depend on the filter and
// sort that find an object's original activity. They skip when no database is reachable.

// spoolSession wraps a real session and spools every post-commit task published through it
type spoolSession struct {
	data.Session
	tasks *postcommit.Tasks
}

// Context carries the post-commit spool, so queued tasks can be counted
func (session spoolSession) Context() context.Context {
	return postcommit.WithContext(session.Session.Context(), session.tasks)
}

// removalRecorder stands in for the legacy outbox and the object cache, recording what each was asked to remove
type removalRecorder struct {
	objectIDs    []string
	activityURLs []string
	cached       []string
}

// removeOutboxMessagesByObjectID records the object whose legacy rows were removed
func (recorder *removalRecorder) removeOutboxMessagesByObjectID(_ data.Session, _ string, _ primitive.ObjectID, objectID string) error {
	recorder.objectIDs = append(recorder.objectIDs, objectID)
	return nil
}

// removeOutboxMessagesByActivityURL records the activity whose legacy row was removed
func (recorder *removalRecorder) removeOutboxMessagesByActivityURL(_ data.Session, _ string, _ primitive.ObjectID, activityURL string) error {
	recorder.activityURLs = append(recorder.activityURLs, activityURL)
	return nil
}

// Delete records the document that was removed from the cache
func (recorder *removalRecorder) Delete(url string) error {
	recorder.cached = append(recorder.cached, url)
	return nil
}

// outbox2FollowUpTest holds an Outbox2 on example.com, its session, and what it removed elsewhere
type outbox2FollowUpTest struct {
	service  *Outbox2
	session  data.Session
	tasks    *postcommit.Tasks
	removals *removalRecorder
	userID   primitive.ObjectID
}

// newOutbox2FollowUpTest returns an Outbox2 service that can publish, delete and undo for one User
func newOutbox2FollowUpTest(t *testing.T) outbox2FollowUpTest {

	t.Helper()

	query := newOutbox2QueryTest(t)
	tasks := postcommit.NewTasks()
	removals := &removalRecorder{}

	service := &Outbox2{
		locator:      &Locator{host: "https://example.com"},
		host:         "https://example.com",
		legacyOutbox: removals,
		objectCache:  removals,
	}

	return outbox2FollowUpTest{
		service:  service,
		session:  spoolSession{Session: query.session, tasks: tasks},
		tasks:    tasks,
		removals: removals,
		userID:   primitive.NewObjectID(),
	}
}

// actor returns the URL of the test's User
func (test outbox2FollowUpTest) actor() string {
	return "https://example.com/@" + test.userID.Hex()
}

// followers returns the URL of the test User's followers collection
func (test outbox2FollowUpTest) followers() string {
	return test.actor() + "/pub/followers"
}

// publish stores an activity of the given type about the given object, addressed as provided
func (test outbox2FollowUpTest) publish(t *testing.T, activityType string, objectID string, permissions model.Permissions, addressing mapof.Any) string {

	t.Helper()

	activity := mapof.Any{
		vocab.PropertyType:   activityType,
		vocab.PropertyActor:  test.actor(),
		vocab.PropertyObject: mapof.Any{vocab.PropertyID: objectID, vocab.PropertyType: vocab.ObjectTypeNote},
	}

	for name, value := range addressing {
		activity[name] = value
	}

	activityURL, err := test.service.Publish(test.session, model.ActorTypeUser, test.userID, streams.NewDocument(activity), permissions)
	require.NoError(t, err)

	return activityURL
}

// only returns the single row of the given type in the test User's outbox
func (test outbox2FollowUpTest) only(t *testing.T, activityType string) model.OutboxItem {

	t.Helper()

	criteria := exp.Equal("actorId", test.userID).AndEqual("activityType", activityType)
	items, err := test.service.Query(test.session, criteria)
	require.NoError(t, err)
	require.Len(t, items, 1, activityType)

	return items[0]
}

// count returns the number of rows of the given type in the test User's outbox
func (test outbox2FollowUpTest) count(t *testing.T, activityType string) int {

	t.Helper()

	items, err := test.service.Query(test.session, exp.Equal("actorId", test.userID).AndEqual("activityType", activityType))
	require.NoError(t, err)

	return len(items)
}

// TestOutbox2_DeleteActivity_CopiesPrivateAddressing verifies that deleting a direct message tells
// only the people it was sent to, and removes it from both outboxes and the cache
func TestOutbox2_DeleteActivity_CopiesPrivateAddressing(t *testing.T) {

	test := newOutbox2FollowUpTest(t)
	permissions := model.NewPermissions()

	const object = "https://example.com/notes/1"
	const bob = "https://remote.example/@bob"
	const carol = "https://remote.example/@carol"

	test.publish(t, vocab.ActivityTypeCreate, object, permissions, mapof.Any{
		vocab.PropertyTo:  []string{bob},
		vocab.PropertyBCC: []string{carol},
	})
	test.tasks.Drain()

	require.NoError(t, test.service.DeleteActivity(test.session, model.ActorTypeUser, test.userID, object, permissions, nil))

	// The Create is gone from both outboxes, and the object from the cache
	require.Zero(t, test.count(t, vocab.ActivityTypeCreate))
	require.Equal(t, []string{object}, test.removals.objectIDs)
	require.Equal(t, []string{object}, test.removals.cached)

	// The Delete names the object, and only the original recipients
	deleted := test.only(t, vocab.ActivityTypeDelete)
	require.Equal(t, object, deleted.ObjectID)
	require.Equal(t, test.actor(), deleted.Activity.GetString(vocab.PropertyActor))
	require.Equal(t, []string{bob}, deleted.Activity.GetSliceOfString(vocab.PropertyTo))
	require.Equal(t, []string{carol}, deleted.Activity.GetSliceOfString(vocab.PropertyBCC))
	require.Empty(t, deleted.Activity.GetSliceOfString(vocab.PropertyCC), "a private Delete must not name the followers")
	require.ElementsMatch(t, []string{bob, carol}, []string(deleted.Recipients))
	require.NotEmpty(t, deleted.URL, "Publish mints the Delete's id")

	// One task fans the Delete out, and carries it with its blind recipients
	tasks := test.tasks.Drain()
	require.Len(t, tasks, 1)
	require.Equal(t, delivery.TaskFanOut, tasks[0].Name)

	queued := tasks[0].Arguments.GetMap("activity")
	require.Equal(t, deleted.URL, queued.GetString(vocab.PropertyID))
	require.Equal(t, []string{carol}, queued.GetSliceOfString(vocab.PropertyBCC))
}

// TestOutbox2_DeleteActivity_PublicAddsFollowers verifies that the Delete of a public object also
// names the followers collection, once
func TestOutbox2_DeleteActivity_PublicAddsFollowers(t *testing.T) {

	const mentioned = "https://remote.example/@mentioned"

	t.Run("followers missing", func(t *testing.T) {

		test := newOutbox2FollowUpTest(t)
		permissions := model.NewAnonymousPermissions()
		object := "https://example.com/notes/2"

		test.publish(t, vocab.ActivityTypeCreate, object, permissions, mapof.Any{
			vocab.PropertyTo: []string{vocab.NamespacePublic},
			vocab.PropertyCC: []string{mentioned},
		})

		require.NoError(t, test.service.DeleteActivity(test.session, model.ActorTypeUser, test.userID, object, permissions, nil))

		deleted := test.only(t, vocab.ActivityTypeDelete)
		require.Equal(t, []string{vocab.NamespaceActivityStreamsPublic}, deleted.Activity.GetSliceOfString(vocab.PropertyTo))
		require.Equal(t, []string{mentioned, test.followers()}, deleted.Activity.GetSliceOfString(vocab.PropertyCC))
	})

	t.Run("followers already named", func(t *testing.T) {

		test := newOutbox2FollowUpTest(t)
		permissions := model.NewAnonymousPermissions()
		object := "https://example.com/notes/3"

		test.publish(t, vocab.ActivityTypeCreate, object, permissions, mapof.Any{
			vocab.PropertyTo: []string{test.followers()},
			vocab.PropertyCC: []string{vocab.NamespacePublic},
		})

		require.NoError(t, test.service.DeleteActivity(test.session, model.ActorTypeUser, test.userID, object, permissions, nil))

		deleted := test.only(t, vocab.ActivityTypeDelete)
		require.Equal(t, []string{test.followers()}, deleted.Activity.GetSliceOfString(vocab.PropertyTo))
		require.Equal(t, []string{vocab.NamespaceActivityStreamsPublic}, deleted.Activity.GetSliceOfString(vocab.PropertyCC))
	})
}

// TestOutbox2_DeleteActivity_UsesNewestPublication verifies that the Delete copies the addressing
// of the newest Create or Update, and ignores an earlier Delete of the same object
func TestOutbox2_DeleteActivity_UsesNewestPublication(t *testing.T) {

	test := newOutbox2FollowUpTest(t)
	permissions := model.NewPermissions()

	const object = "https://example.com/notes/4"
	const first = "https://remote.example/@first"
	const second = "https://remote.example/@second"

	saved := newOutbox2QueryTest(t)
	saved.session = test.session

	withActivity := func(activityType string, recipient string) func(*model.OutboxItem) {
		return func(item *model.OutboxItem) {
			item.ObjectID = object
			item.ActivityType = activityType
			item.Activity = mapof.Any{vocab.PropertyTo: []string{recipient}}
		}
	}

	saved.save(t, model.ActorTypeUser, test.userID, 100, permissions, withActivity(vocab.ActivityTypeCreate, first))
	saved.save(t, model.ActorTypeUser, test.userID, 200, permissions, withActivity(vocab.ActivityTypeUpdate, second))
	saved.save(t, model.ActorTypeUser, test.userID, 300, permissions, withActivity(vocab.ActivityTypeDelete, "https://remote.example/@stale"))

	require.NoError(t, test.service.DeleteActivity(test.session, model.ActorTypeUser, test.userID, object, permissions, nil))

	// Every earlier row for the object is gone, the stale Delete included
	require.Zero(t, test.count(t, vocab.ActivityTypeCreate))
	require.Zero(t, test.count(t, vocab.ActivityTypeUpdate))

	deleted := test.only(t, vocab.ActivityTypeDelete)
	require.Equal(t, []string{second}, deleted.Activity.GetSliceOfString(vocab.PropertyTo))
}

// TestOutbox2_DeleteActivity_WithoutOriginal verifies the addressing of a Delete whose object was
// published through the legacy outbox, which stored no addressing
func TestOutbox2_DeleteActivity_WithoutOriginal(t *testing.T) {

	t.Run("public", func(t *testing.T) {

		test := newOutbox2FollowUpTest(t)

		audience := mapof.Any{vocab.PropertyTo: []string{vocab.NamespacePublic}, vocab.PropertyCC: []string{test.followers()}}
		require.NoError(t, test.service.DeleteActivity(test.session, model.ActorTypeUser, test.userID, "https://example.com/notes/5", model.NewAnonymousPermissions(), audience))

		deleted := test.only(t, vocab.ActivityTypeDelete)
		require.Equal(t, []string{vocab.NamespaceActivityStreamsPublic}, deleted.Activity.GetSliceOfString(vocab.PropertyTo))
		require.Equal(t, []string{test.followers()}, deleted.Activity.GetSliceOfString(vocab.PropertyCC))
		require.Equal(t, []string{"https://example.com/notes/5"}, test.removals.objectIDs, "the legacy rows are still removed")
	})

	t.Run("restricted", func(t *testing.T) {

		test := newOutbox2FollowUpTest(t)
		permissions := model.Permissions{primitive.NewObjectID()}

		require.NoError(t, test.service.DeleteActivity(test.session, model.ActorTypeUser, test.userID, "https://example.com/notes/6", permissions, mapof.NewAny()))

		deleted := test.only(t, vocab.ActivityTypeDelete)
		require.Empty(t, deleted.Recipients, "an empty audience names no one, and never the followers")
		require.Equal(t, permissions, deleted.Permissions)
	})

	t.Run("circle", func(t *testing.T) {

		test := newOutbox2FollowUpTest(t)
		circleURL := test.actor() + "/pub/circles/" + primitive.NewObjectID().Hex()
		permissions := model.Permissions{primitive.NewObjectID()}

		audience := mapof.Any{vocab.PropertyTo: []string{circleURL}}
		require.NoError(t, test.service.DeleteActivity(test.session, model.ActorTypeUser, test.userID, "https://example.com/notes/9", permissions, audience))

		deleted := test.only(t, vocab.ActivityTypeDelete)
		require.Equal(t, []string{circleURL}, deleted.Activity.GetSliceOfString(vocab.PropertyTo), "the circle that received it hears of its deletion")
		require.Empty(t, deleted.Activity.GetSliceOfString(vocab.PropertyCC), "and no followers are added to a restricted audience")
	})
}

// TestOutbox2_DeleteActivity_Refuses verifies that a Delete with no actor URL or no object changes nothing
func TestOutbox2_DeleteActivity_Refuses(t *testing.T) {

	test := newOutbox2FollowUpTest(t)
	permissions := model.NewAnonymousPermissions()

	require.Error(t, test.service.DeleteActivity(test.session, model.ActorTypeSearchQuery, test.userID, "https://example.com/notes/7", permissions, nil))
	require.Error(t, test.service.DeleteActivity(test.session, model.ActorTypeUser, test.userID, "", permissions, nil))

	require.Zero(t, countOutbox2(t, test.session))
	require.Empty(t, test.removals.objectIDs)
	require.Empty(t, test.removals.cached)
	require.Empty(t, test.tasks.Drain())
}

// TestOutbox2_UndoActivity verifies that an Undo removes the original from both outboxes, embeds
// it without its blind recipients, and reaches whoever received the original
func TestOutbox2_UndoActivity(t *testing.T) {

	test := newOutbox2FollowUpTest(t)
	permissions := model.NewAnonymousPermissions()

	const author = "https://remote.example/@author"
	const blind = "https://remote.example/@blind"
	likeURL := test.actor() + "/pub/liked/" + primitive.NewObjectID().Hex()

	like := mapof.Any{
		vocab.PropertyID:     likeURL,
		vocab.PropertyType:   vocab.ActivityTypeLike,
		vocab.PropertyActor:  test.actor(),
		vocab.PropertyObject: "https://remote.example/notes/1",
		vocab.PropertyTo:     []string{author},
		vocab.PropertyBCC:    []string{blind},
	}

	// Publish a copy, so that the caller's map below is the one the Undo is built from
	_, err := test.service.Publish(test.session, model.ActorTypeUser, test.userID, streams.NewDocument(like).Clone(), permissions)
	require.NoError(t, err)
	require.Equal(t, 1, test.count(t, vocab.ActivityTypeLike))
	test.tasks.Drain()

	require.NoError(t, test.service.UndoActivity(test.session, model.ActorTypeUser, test.userID, like, permissions))

	// The Like is gone from both outboxes
	require.Zero(t, test.count(t, vocab.ActivityTypeLike))
	require.Equal(t, []string{likeURL}, test.removals.activityURLs)

	// The Undo embeds the Like, without its blind recipients
	undo := test.only(t, vocab.ActivityTypeUndo)
	require.Equal(t, likeURL, undo.ObjectID)
	require.NotEqual(t, likeURL, undo.URL, "the Undo has an id of its own")

	embedded := streams.NewDocument(undo.Activity).Object()
	require.Equal(t, vocab.ActivityTypeLike, embedded.Type())
	require.Equal(t, likeURL, embedded.ID())
	require.NotContains(t, embedded.Map(), vocab.PropertyBCC)

	// The Undo itself reaches the same people, blind recipients included
	require.Equal(t, []string{author}, undo.Activity.GetSliceOfString(vocab.PropertyTo))
	require.Equal(t, []string{blind}, undo.Activity.GetSliceOfString(vocab.PropertyBCC))
	require.ElementsMatch(t, []string{author, blind}, []string(undo.Recipients))

	// The caller's activity is unchanged
	require.Equal(t, []string{blind}, like[vocab.PropertyBCC])

	require.Len(t, test.tasks.Drain(), 1)
}

// TestOutbox2_UndoActivity_RowAlreadyGone verifies that an Undo is still published when the
// original's row was deleted first, as a Block's is
func TestOutbox2_UndoActivity_RowAlreadyGone(t *testing.T) {

	test := newOutbox2FollowUpTest(t)

	block := mapof.Any{
		vocab.PropertyID:     test.actor() + "/pub/blocked/" + primitive.NewObjectID().Hex(),
		vocab.PropertyType:   vocab.ActivityTypeBlock,
		vocab.PropertyActor:  test.actor(),
		vocab.PropertyObject: "https://remote.example/@blocked",
	}

	require.NoError(t, test.service.UndoActivity(test.session, model.ActorTypeUser, test.userID, block, model.NewAnonymousPermissions()))

	undo := test.only(t, vocab.ActivityTypeUndo)
	require.Equal(t, vocab.ActivityTypeBlock, streams.NewDocument(undo.Activity).Object().Type())
	require.Empty(t, undo.Recipients, "an activity that named no one makes an Undo that names no one")
}

// TestOutbox2_UndoActivity_Refuses verifies that an Undo with no actor URL or no original id changes nothing
func TestOutbox2_UndoActivity_Refuses(t *testing.T) {

	test := newOutbox2FollowUpTest(t)
	permissions := model.NewAnonymousPermissions()
	like := mapof.Any{vocab.PropertyID: "https://example.com/likes/1", vocab.PropertyType: vocab.ActivityTypeLike}

	require.Error(t, test.service.UndoActivity(test.session, model.ActorTypeApplication, test.userID, like, permissions))
	require.Error(t, test.service.UndoActivity(test.session, model.ActorTypeUser, test.userID, mapof.Any{vocab.PropertyType: vocab.ActivityTypeLike}, permissions))

	require.Zero(t, countOutbox2(t, test.session))
	require.Empty(t, test.removals.activityURLs)
	require.Empty(t, test.tasks.Drain())
}

// TestOutbox2_DeleteActivity_StreamActor verifies that a Stream actor's Delete names the Stream's own URL and followers
func TestOutbox2_DeleteActivity_StreamActor(t *testing.T) {

	test := newOutbox2FollowUpTest(t)
	streamID := primitive.NewObjectID()
	streamURL := "https://example.com/" + streamID.Hex()

	audience := mapof.Any{vocab.PropertyTo: []string{vocab.NamespacePublic}, vocab.PropertyCC: []string{streamURL + "/pub/followers"}}
	require.NoError(t, test.service.DeleteActivity(test.session, model.ActorTypeStream, streamID, "https://example.com/notes/8", model.NewAnonymousPermissions(), audience))

	items, err := test.service.Query(test.session, exp.Equal("actorId", streamID))
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, streamURL, items[0].Activity.GetString(vocab.PropertyActor))
	require.Equal(t, []string{streamURL + "/pub/followers"}, items[0].Activity.GetSliceOfString(vocab.PropertyCC))
	require.Equal(t, streamURL+"/pub/outbox/"+items[0].ActivityID.Hex(), items[0].URL)
}

// TestOutbox2_PublishAnnounce verifies that a Stream's boost is stored in its outbox under a URL
// that can be fetched, addressed to Public and the Stream's followers, and fanned out once
func TestOutbox2_PublishAnnounce(t *testing.T) {

	test := newOutbox2FollowUpTest(t)
	streamID := primitive.NewObjectID()
	streamURL := "https://example.com/" + streamID.Hex()

	note := streams.NewDocument(mapof.Any{
		vocab.PropertyID:      "https://remote.example/notes/1",
		vocab.PropertyType:    vocab.ObjectTypeNote,
		vocab.PropertyTo:      []any{"https://remote.example/@carol"},
		vocab.PropertyContent: "Hello",
	})

	activityURL, err := test.service.PublishAnnounce(test.session, model.ActorTypeStream, streamID, streamURL, note)
	require.NoError(t, err)
	require.Equal(t, streamURL+"/pub/outbox/", activityURL[:len(streamURL)+len("/pub/outbox/")], "the Stream's outbox route serves this URL")

	// The row is the Stream's, public, and about the Note
	items, err := test.service.Query(test.session, exp.Equal("actorId", streamID))
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, model.ActorTypeStream, items[0].ActorType)
	require.Equal(t, vocab.ActivityTypeAnnounce, items[0].ActivityType)
	require.Equal(t, "https://remote.example/notes/1", items[0].ObjectID)
	require.True(t, items[0].Permissions.IsAnonymous())

	// One fan-out carries it to Public and the Stream's followers, never to the Note's own audience
	tasks := test.tasks.Drain()
	require.Len(t, tasks, 1)
	require.Equal(t, delivery.TaskFanOut, tasks[0].Name)
	require.Equal(t, streamURL, tasks[0].Arguments.GetString("actor"))

	activity := tasks[0].Arguments.GetMap("activity")
	require.Equal(t, activityURL, activity.GetString(vocab.PropertyID))
	require.Equal(t, []string{vocab.NamespaceActivityStreamsPublic}, activity.GetSliceOfString(vocab.PropertyTo))
	require.Equal(t, []string{streamURL + "/pub/followers"}, activity.GetSliceOfString(vocab.PropertyCC))
}

// TestOutbox2_PublishAnnounce_RemovedByDelete verifies that a Delete of a boosted Note finds the
// Stream's Announce of it, which is how the Stream's Delete handler looks it up
func TestOutbox2_PublishAnnounce_RemovedByDelete(t *testing.T) {

	test := newOutbox2FollowUpTest(t)
	streamID := primitive.NewObjectID()
	streamURL := "https://example.com/" + streamID.Hex()

	boost := func(noteURL string) {
		note := streams.NewDocument(mapof.Any{vocab.PropertyID: noteURL, vocab.PropertyType: vocab.ObjectTypeNote, vocab.PropertyContent: "Hello"})
		_, err := test.service.PublishAnnounce(test.session, model.ActorTypeStream, streamID, streamURL, note)
		require.NoError(t, err)
	}

	boost("https://remote.example/notes/1")
	boost("https://remote.example/notes/2")

	// The inbound Delete names the Note, and that is the key the row was stored under
	require.NoError(t, test.service.RemoveByObjectID(test.session, model.ActorTypeStream, streamID, "https://remote.example/notes/1"))

	items, err := test.service.Query(test.session, exp.Equal("actorId", streamID))
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "https://remote.example/notes/2", items[0].ObjectID)
}

// TestOutbox2_UndoActivity_ObjectRecipients verifies that an Undo reaches a recipient whom the
// original named as an object, and that the recipient can read the Undo back
func TestOutbox2_UndoActivity_ObjectRecipients(t *testing.T) {

	// O3: copyAddressing read addressing as strings, so a recipient written as an object was
	// dropped from the Undo, and the original's author was never told it was withdrawn
	test := newOutbox2FollowUpTest(t)
	permissions := model.NewPermissions()

	const author = "https://remote.example/@author"
	likeURL := test.actor() + "/pub/liked/" + primitive.NewObjectID().Hex()

	like := mapof.Any{
		vocab.PropertyID:     likeURL,
		vocab.PropertyType:   vocab.ActivityTypeLike,
		vocab.PropertyActor:  test.actor(),
		vocab.PropertyObject: "https://remote.example/notes/1",
		vocab.PropertyTo:     []any{mapof.Any{vocab.PropertyID: author, vocab.PropertyType: vocab.ActorTypePerson}},
	}

	require.NoError(t, test.service.UndoActivity(test.session, model.ActorTypeUser, test.userID, like, permissions))

	undo := test.only(t, vocab.ActivityTypeUndo)
	require.Equal(t, []string{author}, undo.Activity.GetSliceOfString(vocab.PropertyTo))
	require.Equal(t, []string{author}, []string(undo.Recipients))
	require.True(t, undo.IsVisibleTo(permissions, author))
}

// TestCopyAddressing verifies that every addressing property is copied by id, that a property the
// source lacks is left alone, and that the target never shares a slice with the source
func TestCopyAddressing(t *testing.T) {

	source := mapof.Any{
		vocab.PropertyTo:   []any{mapof.Any{vocab.PropertyID: "https://a.example/1"}, "https://a.example/2", mapof.Any{vocab.PropertyType: "Person"}},
		vocab.PropertyCC:   "https://a.example/3",
		vocab.PropertyBCC:  []string{"https://a.example/4"},
		vocab.PropertyType: vocab.ActivityTypeCreate,
	}

	target := mapof.Any{vocab.PropertyBTo: []string{"https://kept.example/1"}}
	copyAddressing(target, source)

	require.Equal(t, []string{"https://a.example/1", "https://a.example/2"}, target[vocab.PropertyTo])
	require.Equal(t, []string{"https://a.example/3"}, target[vocab.PropertyCC])
	require.Equal(t, []string{"https://a.example/4"}, target[vocab.PropertyBCC])
	require.Equal(t, []string{"https://kept.example/1"}, target[vocab.PropertyBTo], "a property the source lacks is left alone")
	require.NotContains(t, target, vocab.PropertyType, "only addressing is copied")

	// Changing the copy leaves the source as it was
	target[vocab.PropertyBCC].([]string)[0] = "https://changed.example/1"
	require.Equal(t, []string{"https://a.example/4"}, source[vocab.PropertyBCC])

	// A nil or empty source copies nothing
	empty := mapof.Any{}
	copyAddressing(empty, nil)
	copyAddressing(empty, mapof.Any{vocab.PropertyTo: []any{}})
	require.Empty(t, empty)
}

// TestAddFollowersToPublic verifies that followers are added to `cc` only when the addressing
// names Public, however Public is written, and never twice
func TestAddFollowersToPublic(t *testing.T) {

	const followers = "https://example.com/@user/pub/followers"

	// test checks the `cc` that results from one addressing
	test := func(name string, addressing mapof.Any, expected []string) {
		t.Run(name, func(t *testing.T) {
			addFollowersToPublic(addressing, followers)
			require.Equal(t, expected, addressIDs(addressing, vocab.PropertyCC))
		})
	}

	test("public adds followers", mapof.Any{vocab.PropertyTo: []string{vocab.NamespaceActivityStreamsPublic}}, []string{followers})
	test("short public adds followers", mapof.Any{vocab.PropertyTo: "Public"}, []string{followers})
	test("public as an object adds followers", mapof.Any{vocab.PropertyTo: []any{mapof.Any{vocab.PropertyID: vocab.NamespaceActivityStreamsPublic}}}, []string{followers})
	test("public in cc keeps cc", mapof.Any{vocab.PropertyCC: []any{mapof.Any{vocab.PropertyID: "as:Public"}, "https://b.example/1"}}, []string{vocab.NamespaceActivityStreamsPublic, "https://b.example/1", followers})
	test("followers already named are not added twice", mapof.Any{vocab.PropertyTo: "as:Public", vocab.PropertyBCC: []any{mapof.Any{vocab.PropertyID: followers}}}, []string{})
	test("private addressing adds no one", mapof.Any{vocab.PropertyTo: []string{"https://b.example/1"}}, []string{})
	test("no addressing adds no one", mapof.Any{}, []string{})
}

// TestReplaceActorInAddressing verifies that an imported activity's addressing is pointed at the
// local actor, including a recipient written as an object
func TestReplaceActorInAddressing(t *testing.T) {

	const remote = "https://old.example/@ben"
	const local = "https://new.example/@0123456789abcdef01234567"

	activity := mapof.Any{
		vocab.PropertyTo:  []any{mapof.Any{vocab.PropertyID: remote + "/pub/followers"}, "https://friend.example/@a"},
		vocab.PropertyCC:  remote,
		vocab.PropertyBCC: []any{mapof.Any{vocab.PropertyType: "Person"}},
	}

	replaceActorInAddressing(activity, remote, local)

	require.Equal(t, []string{local + "/pub/followers", "https://friend.example/@a"}, activity[vocab.PropertyTo])
	require.Equal(t, []string{local}, activity[vocab.PropertyCC])
	require.Equal(t, []any{mapof.Any{vocab.PropertyType: "Person"}}, activity[vocab.PropertyBCC], "a property that names no one is left as it was")

	// RULE: An empty remote actor would match every recipient, so nothing changes
	unchanged := mapof.Any{vocab.PropertyTo: []string{"https://friend.example/@a"}}
	replaceActorInAddressing(unchanged, "", local)
	require.Equal(t, []string{"https://friend.example/@a"}, unchanged[vocab.PropertyTo])
}

// TestAddressIDs_WritesPublicAsTheFullIRI verifies that every spelling of Public is read as the
// full IRI, and that nothing else is rewritten
func TestAddressIDs_WritesPublicAsTheFullIRI(t *testing.T) {

	// O6: GoToSocial, Pixelfed, PeerTube, Lemmy 0.19, Friendica and Fedify do not read a bare
	// "Public" as public, and a Delete or Undo copies its addressing from rows stored with one
	test := func(name string, recipients any, expected []string) {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, expected, addressIDs(mapof.Any{vocab.PropertyTo: recipients}, vocab.PropertyTo))
		})
	}

	test("bare", "Public", []string{vocab.NamespaceActivityStreamsPublic})
	test("compact", "as:Public", []string{vocab.NamespaceActivityStreamsPublic})
	test("full", vocab.NamespaceActivityStreamsPublic, []string{vocab.NamespaceActivityStreamsPublic})
	test("as an object", []any{mapof.Any{vocab.PropertyID: "Public"}}, []string{vocab.NamespaceActivityStreamsPublic})
	test("among others, in place", []any{"https://a.example/1", "Public", "https://a.example/2"}, []string{"https://a.example/1", vocab.NamespaceActivityStreamsPublic, "https://a.example/2"})
	test("lowercase is not Public", "public", []string{"public"})
	test("a URL ending in Public is not Public", "https://a.example/Public", []string{"https://a.example/Public"})
	test("http is not Public", "http://www.w3.org/ns/activitystreams#Public", []string{"http://www.w3.org/ns/activitystreams#Public"})

	// The source keeps its own spelling, so a stored row is never rewritten by reading it
	source := mapof.Any{vocab.PropertyTo: []string{"Public"}}
	addressIDs(source, vocab.PropertyTo)
	require.Equal(t, []string{"Public"}, source[vocab.PropertyTo])
}

// FuzzAddressIDs verifies that any JSON addressing yields ids without panicking, that none is
// empty or a short spelling of Public, and that copying it twice gives the same result
func FuzzAddressIDs(f *testing.F) {

	f.Add(`{"to":["https://a.example/1",{"id":"https://a.example/2"}],"cc":"as:Public"}`)
	f.Add(`{"to":[{"type":"Person"},null,1,true,[["x"]]],"bcc":{"id":{"id":"y"}}}`)
	f.Add(`{"to":"","cc":[""]}`)
	f.Add(`{"to":["Public",{"id":"as:Public"}],"bto":"https://www.w3.org/ns/activitystreams#Public"}`)
	f.Add(`null`)

	f.Fuzz(func(t *testing.T, document string) {

		source := mapof.NewAny()

		if err := json.Unmarshal([]byte(document), &source); err != nil {
			return
		}

		first := mapof.Any{}
		copyAddressing(first, source)

		// RULE: No id is empty, and Public is only ever written as the full IRI
		for _, name := range addressingProperties {
			for _, id := range addressIDs(source, name) {
				require.NotEmpty(t, id)
				require.NotContains(t, []string{vocab.NamespacePublic, vocab.NamespaceASPublic}, id)
			}
		}

		// Copying the copy changes nothing, so a Delete of a Delete is addressed the same way
		second := mapof.Any{}
		copyAddressing(second, first)
		require.Equal(t, first, second)

		addFollowersToPublic(first, "https://example.com/@user/pub/followers")
	})
}
