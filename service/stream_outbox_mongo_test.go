package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// These tests run Stream.Save against a real MongoDB, because the rule they guard compares the Stream
// being saved with the copy in the database, which shares no memory with it.

// outboxTestStates gives the test collection a draft State, a State that is sent to followers, and a second of each
func outboxTestStates() mapof.Object[model.State] {
	return mapof.Object[model.State]{
		"default":   {StateID: "default"},
		"hidden":    {StateID: "hidden"},
		"published": {StateID: "published", IsOutbox: true},
		"featured":  {StateID: "featured", IsOutbox: true},
	}
}

// newOutboxSaveTest returns a Stream service whose test collection has outboxTestStates, a session on a
// throwaway database, and the spool that collects the tasks its saves queue
func newOutboxSaveTest(t *testing.T) (*Stream, data.Session, *postcommit.Tasks) {

	t.Helper()

	database := newStreamTokenDatabase(t)
	tasks := postcommit.NewTasks()

	session, err := mongodb.NewServer(database).Session(postcommit.WithContext(t.Context(), tasks))
	require.NoError(t, err)

	service := newStreamTokenService()
	setOutboxTestStates(service, outboxTestStates())

	return service, session, tasks
}

// setOutboxTestStates replaces the States of the test collection Template
func setOutboxTestStates(service *Stream, states mapof.Object[model.State]) {
	template := service.templateService.templates["test-collection"]
	template.States = states
	service.templateService.templates["test-collection"] = template
}

// newOutboxStream returns a new Stream on the test collection, in the provided State
func newOutboxStream(stateID string) model.Stream {
	stream := newTokenStream("")
	stream.StateID.Set(stateID)
	return stream
}

// storedStream reads a Stream back from the database, as another server would see it
func storedStream(t *testing.T, service *Stream, session data.Session, streamID primitive.ObjectID) model.Stream {

	t.Helper()

	result := model.NewStream()
	require.NoError(t, service.LoadByID(session, streamID, &result))
	return result
}

// drainOutboxTasks returns the StreamIDs of the outbox tasks queued since the last call, and fails on any other task
func drainOutboxTasks(t *testing.T, tasks *postcommit.Tasks) []string {

	t.Helper()

	result := make([]string, 0)

	for _, task := range tasks.Drain() {
		if task.Name == StreamOutboxTaskName {
			result = append(result, task.Arguments.GetString("streamId"))
		}
	}

	return result
}

// TestStream_Save_Outbox_NewStream confirms that a new Stream takes the flag of the State it is created in
func TestStream_Save_Outbox_NewStream(t *testing.T) {

	service, session, tasks := newOutboxSaveTest(t)

	// A draft is outside the outbox, gets no date, and queues nothing
	draft := newOutboxStream("default")
	require.NoError(t, service.Save(session, &draft, "draft"))
	require.False(t, storedStream(t, service, session, draft.StreamID).IsOutbox.Value())
	require.Zero(t, draft.PublishDate, "a Stream outside the outbox has no date")
	require.Empty(t, drainOutboxTasks(t, tasks))

	// A Stream created in an outbox State is in the outbox at once
	note := newOutboxStream("published")
	require.NoError(t, service.Save(session, &note, "note"))

	stored := storedStream(t, service, session, note.StreamID)
	require.True(t, stored.IsOutbox.Value())
	require.Positive(t, stored.PublishDate)
	require.Equal(t, []string{note.StreamID.Hex()}, drainOutboxTasks(t, tasks))
}

// TestStream_Save_Outbox_NewStreamDropsFingerprint confirms that a new record never claims that followers hold it,
// which is what an imported Stream would otherwise carry in
func TestStream_Save_Outbox_NewStreamDropsFingerprint(t *testing.T) {

	service, session, _ := newOutboxSaveTest(t)

	imported := newOutboxStream("published")
	imported.Fingerprint.Set("from-another-server")
	imported.IsOutbox.Set(true)

	require.NoError(t, service.Save(session, &imported, "import"))
	require.Empty(t, imported.Fingerprint)
	require.Empty(t, storedStream(t, service, session, imported.StreamID).Fingerprint)
}

// TestStream_Save_Outbox_Transitions walks a Stream into the outbox, through an edit, and out again
func TestStream_Save_Outbox_Transitions(t *testing.T) {

	service, session, tasks := newOutboxSaveTest(t)

	stream := newOutboxStream("default")
	require.NoError(t, service.Save(session, &stream, "created"))
	require.Empty(t, drainOutboxTasks(t, tasks))

	// A save that keeps the draft State queues nothing
	stream.Label = "Edited draft"
	require.NoError(t, service.Save(session, &stream, "edited"))
	require.False(t, stream.IsOutbox.Value())
	require.Empty(t, drainOutboxTasks(t, tasks))

	// Entering an outbox State raises the flag, dates the Stream, and queues the task
	stream.StateID.Set("published")
	require.NoError(t, service.Save(session, &stream, "published"))
	require.True(t, storedStream(t, service, session, stream.StreamID).IsOutbox.Value())
	require.Positive(t, stream.PublishDate)
	require.Equal(t, []string{stream.StreamID.Hex()}, drainOutboxTasks(t, tasks))

	// A save inside the outbox queues the task again, to decide whether followers need an Update
	stream.Label = "Edited note"
	require.NoError(t, service.Save(session, &stream, "edited"))
	require.True(t, stream.IsOutbox.Value())
	require.Len(t, drainOutboxTasks(t, tasks), 1)

	// Moving between two outbox States keeps the flag
	stream.StateID.Set("featured")
	require.NoError(t, service.Save(session, &stream, "featured"))
	require.True(t, stream.IsOutbox.Value())
	require.Len(t, drainOutboxTasks(t, tasks), 1)

	// Leaving the outbox clears the flag, and queues the task that sends the Delete
	stream.StateID.Set("hidden")
	require.NoError(t, service.Save(session, &stream, "hidden"))
	require.False(t, storedStream(t, service, session, stream.StreamID).IsOutbox.Value())
	require.Len(t, drainOutboxTasks(t, tasks), 1)

	// After that, with no fingerprint, a save queues nothing
	stream.Label = "Hidden and edited"
	require.NoError(t, service.Save(session, &stream, "edited"))
	require.Empty(t, drainOutboxTasks(t, tasks))
}

// TestStream_Save_Outbox_PublishDateIsSetOnce confirms that a Stream keeps its date when it enters the outbox
// a second time, and that a backdated Stream keeps the date it was given
func TestStream_Save_Outbox_PublishDateIsSetOnce(t *testing.T) {

	service, session, _ := newOutboxSaveTest(t)

	// A backdated Stream keeps its date on entry
	backdated := newOutboxStream("default")
	backdated.PublishDate = 1000
	require.NoError(t, service.Save(session, &backdated, "created"))

	backdated.StateID.Set("published")
	require.NoError(t, service.Save(session, &backdated, "published"))
	require.Equal(t, int64(1000), backdated.PublishDate)

	// ...and again after leaving and coming back
	backdated.StateID.Set("default")
	require.NoError(t, service.Save(session, &backdated, "unpublished"))
	backdated.StateID.Set("published")
	require.NoError(t, service.Save(session, &backdated, "republished"))
	require.Equal(t, int64(1000), storedStream(t, service, session, backdated.StreamID).PublishDate)

	// A Stream with no date is dated on entry
	stream := newOutboxStream("default")
	require.NoError(t, service.Save(session, &stream, "created"))
	require.Zero(t, stream.PublishDate)

	stream.StateID.Set("published")
	require.NoError(t, service.Save(session, &stream, "published"))
	require.Positive(t, stream.PublishDate)
}

// TestStream_Save_Outbox_TemplateEditMovesNothing confirms that changing a State's flag in the Template leaves
// an existing Stream alone until its State changes
func TestStream_Save_Outbox_TemplateEditMovesNothing(t *testing.T) {

	service, session, tasks := newOutboxSaveTest(t)

	sent := newOutboxStream("published")
	draft := newOutboxStream("default")
	require.NoError(t, service.Save(session, &sent, "created"))
	require.NoError(t, service.Save(session, &draft, "created"))
	tasks.Reset()

	// Swap the flags: "published" no longer sends, and "default" does
	states := outboxTestStates()
	states["published"] = model.State{StateID: "published"}
	states["default"] = model.State{StateID: "default", IsOutbox: true}
	setOutboxTestStates(service, states)

	// Each Stream keeps the flag it had
	require.NoError(t, service.Save(session, &sent, "saved after the edit"))
	require.NoError(t, service.Save(session, &draft, "saved after the edit"))
	require.True(t, storedStream(t, service, session, sent.StreamID).IsOutbox.Value())
	require.False(t, storedStream(t, service, session, draft.StreamID).IsOutbox.Value())
	require.Equal(t, []string{sent.StreamID.Hex()}, drainOutboxTasks(t, tasks))

	// The new flags apply once a Stream changes State
	draft.StateID.Set("published")
	require.NoError(t, service.Save(session, &draft, "published"))
	require.False(t, draft.IsOutbox.Value())
}

// TestStream_Save_Outbox_IgnoresACallersFlag confirms that setting the flag on a Stream whose State has not
// changed does nothing, because only a State can put a Stream in the outbox
func TestStream_Save_Outbox_IgnoresACallersFlag(t *testing.T) {

	service, session, tasks := newOutboxSaveTest(t)

	stream := newOutboxStream("default")
	require.NoError(t, service.Save(session, &stream, "created"))

	stream.IsOutbox.Set(true)
	require.NoError(t, service.Save(session, &stream, "flag set by hand"))

	require.False(t, stream.IsOutbox.Value())
	require.False(t, storedStream(t, service, session, stream.StreamID).IsOutbox.Value())
	require.Empty(t, drainOutboxTasks(t, tasks))
}

// TestStream_Save_Outbox_FollowersStillHoldIt confirms that a Stream outside the outbox queues the task while
// followers hold a copy, so that the task can send the Delete, and that a save passes the fingerprint through
func TestStream_Save_Outbox_FollowersStillHoldIt(t *testing.T) {

	service, session, tasks := newOutboxSaveTest(t)

	created := newOutboxStream("default")
	require.NoError(t, service.Save(session, &created, "created"))

	// The outbox task records what followers hold
	require.NoError(t, queries.RawUpdate(t.Context(), session.Collection("Stream"), exp.Equal("_id", created.StreamID), bson.M{
		"$set": bson.M{"fingerprint": "claimed"},
	}))

	loaded := storedStream(t, service, session, created.StreamID)
	loaded.Label = "Edited"
	require.NoError(t, service.Save(session, &loaded, "edited"))

	require.Equal(t, "claimed", storedStream(t, service, session, created.StreamID).Fingerprint.Value())
	require.Equal(t, []string{created.StreamID.Hex()}, drainOutboxTasks(t, tasks))
}

// TestStream_Save_Outbox_LoadedCopy confirms that a Stream read back from the database, as each request reads it,
// compares against the State it was loaded in
func TestStream_Save_Outbox_LoadedCopy(t *testing.T) {

	service, session, tasks := newOutboxSaveTest(t)

	created := newOutboxStream("default")
	require.NoError(t, service.Save(session, &created, "created"))

	// A second request publishes it
	second := storedStream(t, service, session, created.StreamID)
	second.StateID.Set("published")
	require.NoError(t, service.Save(session, &second, "published"))
	require.True(t, second.IsOutbox.Value())
	require.Len(t, drainOutboxTasks(t, tasks), 1)

	// A third request unpublishes it
	third := storedStream(t, service, session, created.StreamID)
	require.True(t, third.IsOutbox.Value())
	third.StateID.Set("default")
	require.NoError(t, service.Save(session, &third, "unpublished"))
	require.False(t, storedStream(t, service, session, created.StreamID).IsOutbox.Value())
	require.Len(t, drainOutboxTasks(t, tasks), 1)
}

// TestStream_Save_Outbox_UndefinedState confirms that a State the Template does not define sends nothing
func TestStream_Save_Outbox_UndefinedState(t *testing.T) {

	template := model.NewTemplate("test", nil)
	template.States = outboxTestStates()

	stream := model.NewStream()
	stream.StateID.Set("no-such-state")
	stream.IsOutbox.Set(true)

	applyOutboxState(&template, &stream)
	require.False(t, stream.IsOutbox.Value())
	require.Zero(t, stream.PublishDate)
}

// TestStream_SaveWithoutOutbox confirms that the quiet save stores the Stream and queues nothing, in or out of the outbox
func TestStream_SaveWithoutOutbox(t *testing.T) {

	service, session, tasks := newOutboxSaveTest(t)

	stream := newOutboxStream("published")
	require.NoError(t, service.Save(session, &stream, "created"))
	require.Len(t, drainOutboxTasks(t, tasks), 1)

	stream.ReplyCount = 7
	require.NoError(t, service.SaveWithoutOutbox(session, &stream, "Refreshed reply count"))

	stored := storedStream(t, service, session, stream.StreamID)
	require.Equal(t, 7, stored.ReplyCount)
	require.True(t, stored.IsOutbox.Value())
	require.Empty(t, drainOutboxTasks(t, tasks))
}

// TestStream_Save_Outbox_FingerprintCannotBeSet confirms that a save puts back a fingerprint that the caller
// changed, because only the outbox task's claim writes it
func TestStream_Save_Outbox_FingerprintCannotBeSet(t *testing.T) {

	service, session, _ := newOutboxSaveTest(t)

	created := newOutboxStream("published")
	require.NoError(t, service.Save(session, &created, "created"))

	require.NoError(t, queries.RawUpdate(t.Context(), session.Collection("Stream"), exp.Equal("_id", created.StreamID), bson.M{
		"$set": bson.M{"fingerprint": "claimed"},
	}))

	// Clearing it, or replacing it, has no effect
	for _, forged := range []string{"", "forged"} {
		loaded := storedStream(t, service, session, created.StreamID)
		loaded.Fingerprint.Set(forged)
		require.NoError(t, service.Save(session, &loaded, "forged"))
		require.Equal(t, "claimed", loaded.Fingerprint.Value())
		require.Equal(t, "claimed", storedStream(t, service, session, created.StreamID).Fingerprint.Value())
	}
}
