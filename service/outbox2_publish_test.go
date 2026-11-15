package service

import (
	"context"
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

	require.NoError(t, test.service.DeleteActivity(test.session, model.ActorTypeUser, test.userID, object, permissions))

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

		require.NoError(t, test.service.DeleteActivity(test.session, model.ActorTypeUser, test.userID, object, permissions))

		deleted := test.only(t, vocab.ActivityTypeDelete)
		require.Equal(t, []string{vocab.NamespacePublic}, deleted.Activity.GetSliceOfString(vocab.PropertyTo))
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

		require.NoError(t, test.service.DeleteActivity(test.session, model.ActorTypeUser, test.userID, object, permissions))

		deleted := test.only(t, vocab.ActivityTypeDelete)
		require.Equal(t, []string{test.followers()}, deleted.Activity.GetSliceOfString(vocab.PropertyTo))
		require.Equal(t, []string{vocab.NamespacePublic}, deleted.Activity.GetSliceOfString(vocab.PropertyCC))
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

	require.NoError(t, test.service.DeleteActivity(test.session, model.ActorTypeUser, test.userID, object, permissions))

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

		require.NoError(t, test.service.DeleteActivity(test.session, model.ActorTypeUser, test.userID, "https://example.com/notes/5", model.NewAnonymousPermissions()))

		deleted := test.only(t, vocab.ActivityTypeDelete)
		require.Equal(t, []string{vocab.NamespacePublic}, deleted.Activity.GetSliceOfString(vocab.PropertyTo))
		require.Equal(t, []string{test.followers()}, deleted.Activity.GetSliceOfString(vocab.PropertyCC))
		require.Equal(t, []string{"https://example.com/notes/5"}, test.removals.objectIDs, "the legacy rows are still removed")
	})

	t.Run("restricted", func(t *testing.T) {

		test := newOutbox2FollowUpTest(t)
		permissions := model.Permissions{primitive.NewObjectID()}

		require.NoError(t, test.service.DeleteActivity(test.session, model.ActorTypeUser, test.userID, "https://example.com/notes/6", permissions))

		deleted := test.only(t, vocab.ActivityTypeDelete)
		require.Empty(t, deleted.Recipients, "a restricted object with no stored addressing names no one, and never its followers")
		require.Equal(t, permissions, deleted.Permissions)
	})
}

// TestOutbox2_DeleteActivity_Refuses verifies that a Delete with no actor URL or no object changes nothing
func TestOutbox2_DeleteActivity_Refuses(t *testing.T) {

	test := newOutbox2FollowUpTest(t)
	permissions := model.NewAnonymousPermissions()

	require.Error(t, test.service.DeleteActivity(test.session, model.ActorTypeSearchQuery, test.userID, "https://example.com/notes/7", permissions))
	require.Error(t, test.service.DeleteActivity(test.session, model.ActorTypeUser, test.userID, "", permissions))

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

	require.NoError(t, test.service.DeleteActivity(test.session, model.ActorTypeStream, streamID, "https://example.com/notes/8", model.NewAnonymousPermissions()))

	items, err := test.service.Query(test.session, exp.Equal("actorId", streamID))
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, streamURL, items[0].Activity.GetString(vocab.PropertyActor))
	require.Equal(t, []string{streamURL + "/pub/followers"}, items[0].Activity.GetSliceOfString(vocab.PropertyCC))
	require.Equal(t, streamURL+"/pub/outbox/"+items[0].ActivityID.Hex(), items[0].URL)
}
