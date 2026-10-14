package activitypub_user

import (
	"net/http"
	"testing"

	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestOutbox_ServerAssignedActivityID pins the mechanics putActivityIntoOutbox relies on:
// SetProperty mutates the document's underlying value, so the server-assigned canonical id
// replaces a client-provided one (ActivityPub §6.1) and survives into every later Map() call.
func TestOutbox_ServerAssignedActivityID(t *testing.T) {

	activity := streams.NewDocument(mapof.Any{
		vocab.PropertyType:  vocab.ActivityTypeCreate,
		vocab.PropertyID:    "urn:uuid:118cc497-ca55-42a8-9231-4a54b6d54c00",
		vocab.PropertyActor: "https://example.com/@sender",
		vocab.PropertyTo:    []any{"https://example.com/@sender"},
	})

	const canonicalID = "https://example.com/@sender/pub/outbox/000000000000000000000001"
	activity.SetProperty(vocab.PropertyID, canonicalID)

	require.Equal(t, canonicalID, activity.ID())
	require.Equal(t, canonicalID, activity.Map()[vocab.PropertyID])
}

// TestOutbox_ServerAssignedObjectID pins the mechanics outbox_CreateArticle relies on.
// For an EMBEDDED object, Object().Map() returns the live underlying map (not a clone),
// so stamping the minted object id is immediately visible on the activity. The SetProperty
// set-back keeps the stamp correct in the other wire shape too — a bare string object
// reference, which Map() normalizes into a NEW {id: ...} map that would otherwise be lost.
func TestOutbox_ServerAssignedObjectID(t *testing.T) {

	const objectURL = "https://example.com/@sender/objects/000000000000000000000002"

	// Embedded-object shape: the map aliases, and the set-back is a harmless re-set
	activity := streams.NewDocument(mapof.Any{
		vocab.PropertyType:  vocab.ActivityTypeCreate,
		vocab.PropertyActor: "https://example.com/@sender",
		vocab.PropertyObject: map[string]any{
			vocab.PropertyType:    vocab.ObjectTypeNote,
			vocab.PropertyContent: "Hello",
		},
	})

	objectValue := activity.Object().Map()
	objectValue[vocab.PropertyID] = objectURL
	activity.SetProperty(vocab.PropertyObject, objectValue)

	require.Equal(t, objectURL, activity.Object().ID())
	require.Equal(t, objectURL, streams.NewDocument(activity.Map()).Object().ID())

	// String-reference shape: Map() builds a fresh {id: ...} map, so only the set-back
	// carries the stamp onto the activity
	reference := streams.NewDocument(mapof.Any{
		vocab.PropertyType:   vocab.ActivityTypeCreate,
		vocab.PropertyActor:  "https://example.com/@sender",
		vocab.PropertyObject: "urn:uuid:5f42a834-9d33-4b6c-8f2e-3f0a0e60b57e",
	})

	referenceValue := reference.Object().Map()
	referenceValue[vocab.PropertyID] = objectURL
	reference.SetProperty(vocab.PropertyObject, referenceValue)

	require.Equal(t, objectURL, reference.Object().ID())
}

// TestOutbox_requireUndoActorMatches verifies that an Undo may reverse only its own actor's
// activities, and that an object it cannot read an actor from is left to the receiver
func TestOutbox_requireUndoActorMatches(t *testing.T) {

	// BUG-62: the outbox signed and delivered an Undo of another actor's Like (ActivityPub §6.10)
	const me = "https://example.com/@sender"
	const someoneElse = "https://other.example/@victim"

	// Every document loads through this client, so a test can prove that nothing was fetched
	client := &countingClient{}

	undo := func(object any) streams.Document {
		return streams.NewDocument(mapof.Any{
			vocab.PropertyType:   vocab.ActivityTypeUndo,
			vocab.PropertyActor:  me,
			vocab.PropertyObject: object,
		}, streams.WithClient(client))
	}

	like := func(actor any) map[string]any {
		return map[string]any{
			vocab.PropertyID:     "https://example.com/@sender/pub/liked/1",
			vocab.PropertyType:   vocab.ActivityTypeLike,
			vocab.PropertyActor:  actor,
			vocab.PropertyObject: "https://other.example/notes/1",
		}
	}

	requireForbidden := func(t *testing.T, err error) {
		t.Helper()
		require.Error(t, err)
		require.Equal(t, http.StatusForbidden, derp.ErrorCode(err))
	}

	t.Run("own activity", func(t *testing.T) {
		require.NoError(t, requireUndoActorMatches(undo(like(me))))
	})

	t.Run("own activity, actor embedded", func(t *testing.T) {
		require.NoError(t, requireUndoActorMatches(undo(like(map[string]any{vocab.PropertyID: me}))))
	})

	t.Run("another actor's activity", func(t *testing.T) {
		requireForbidden(t, requireUndoActorMatches(undo(like(someoneElse))))
	})

	t.Run("another actor, embedded", func(t *testing.T) {
		requireForbidden(t, requireUndoActorMatches(undo(like(map[string]any{vocab.PropertyID: someoneElse}))))
	})

	t.Run("one of several actors is someone else", func(t *testing.T) {
		requireForbidden(t, requireUndoActorMatches(undo(like([]any{me, someoneElse}))))
	})

	t.Run("one of several objects is someone else's", func(t *testing.T) {
		requireForbidden(t, requireUndoActorMatches(undo([]any{like(me), like(someoneElse)})))
	})

	t.Run("bare URL object is never fetched", func(t *testing.T) {
		require.NoError(t, requireUndoActorMatches(undo("https://example.com/@sender/pub/liked/1")))
		require.NoError(t, requireUndoActorMatches(undo([]any{"https://example.com/@sender/pub/liked/2", like(me)})))
	})

	t.Run("embedded object with no actor", func(t *testing.T) {
		require.NoError(t, requireUndoActorMatches(undo(map[string]any{vocab.PropertyType: vocab.ActivityTypeLike})))
	})

	t.Run("no object", func(t *testing.T) {
		require.NoError(t, requireUndoActorMatches(streams.NewDocument(mapof.Any{
			vocab.PropertyType:  vocab.ActivityTypeUndo,
			vocab.PropertyActor: me,
		}, streams.WithClient(client))))
	})

	t.Run("not an Undo", func(t *testing.T) {
		announce := undo(like(someoneElse))
		announce.SetProperty(vocab.PropertyType, vocab.ActivityTypeAnnounce)
		require.NoError(t, requireUndoActorMatches(announce))
	})

	require.Zero(t, client.loads, "checking an Undo must never load anything")
}

// countingClient is a streams.Client that counts each Load and refuses it
type countingClient struct {
	loads int
}

// SetRootClient does nothing, because this client wraps no other
func (client *countingClient) SetRootClient(streams.Client) {}

// Load records the attempt and refuses it
func (client *countingClient) Load(uri string, _ ...any) (streams.Document, error) {
	client.loads++
	return streams.NilDocument(), derp.Internal("countingClient.Load", "Loading is not allowed in this test", uri)
}

// Save does nothing, because nothing in these tests writes to the cache
func (client *countingClient) Save(streams.Document) error {
	return nil
}

// Delete does nothing, because nothing in these tests writes to the cache
func (client *countingClient) Delete(string) error {
	return nil
}
