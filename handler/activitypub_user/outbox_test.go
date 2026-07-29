package activitypub_user

import (
	"testing"

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
