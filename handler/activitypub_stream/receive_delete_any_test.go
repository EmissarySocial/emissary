package activitypub_stream

import (
	"testing"

	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestRequireOwnTarget confirms that a Delete or Undo may name only an object on its actor's own origin
func TestRequireOwnTarget(t *testing.T) {

	const actor = "https://good.example/@alice"

	test := func(name string, activityType string, object any, allowed bool) {
		t.Run(name, func(t *testing.T) {

			activity := streams.NewDocument(mapof.Any{
				vocab.PropertyType:   activityType,
				vocab.PropertyActor:  actor,
				vocab.PropertyObject: object,
			})

			err := requireOwnTarget(activity)

			if allowed {
				require.NoError(t, err)
				return
			}

			require.True(t, derp.IsForbidden(err), "want Forbidden, got %v", err)
		})
	}

	test("delete own link", vocab.ActivityTypeDelete, "https://good.example/notes/1", true)
	test("delete own tombstone", vocab.ActivityTypeDelete, mapof.Any{vocab.PropertyID: "https://good.example/notes/1", vocab.PropertyType: vocab.ObjectTypeTombstone}, true)
	test("delete own actor", vocab.ActivityTypeDelete, actor, true)
	test("undo own like", vocab.ActivityTypeUndo, mapof.Any{vocab.PropertyID: "https://good.example/likes/1", vocab.PropertyType: vocab.ActivityTypeLike}, true)
	test("delete foreign link", vocab.ActivityTypeDelete, "https://victim.example/notes/1", false)
	test("delete foreign tombstone", vocab.ActivityTypeDelete, mapof.Any{vocab.PropertyID: "https://victim.example/notes/1"}, false)
	test("delete a lookalike host", vocab.ActivityTypeDelete, "https://good.example.evil.example/notes/1", false)
	test("delete on another port", vocab.ActivityTypeDelete, "https://good.example:8443/notes/1", false)
	test("undo foreign activity", vocab.ActivityTypeUndo, "https://victim.example/announces/1", false)
	test("delete urn:uuid", vocab.ActivityTypeDelete, "urn:uuid:550e8400-e29b-41d4-a716-446655440000", false)
	test("delete object with no id", vocab.ActivityTypeDelete, mapof.Any{vocab.PropertyType: vocab.ObjectTypeTombstone}, false)
	test("delete nothing", vocab.ActivityTypeDelete, nil, false)
}

// TestDeleteAny_RefusesForeignObjectBeforeAnyLookup confirms that a Stream refuses a Delete of an
// object on another origin, and does so before it reaches the factory
func TestDeleteAny_RefusesForeignObjectBeforeAnyLookup(t *testing.T) {

	// O14: any signed actor could name a local child Stream, remove the parent's Announce of it,
	// and have the parent rebroadcast the Delete to every follower. The Context has no factory,
	// so any lookup before the check would nil-panic the test.
	for _, activityType := range []string{vocab.ActivityTypeDelete, vocab.ActivityTypeUndo} {

		activity := streams.NewDocument(mapof.Any{
			vocab.PropertyID:     "https://mallory.example/activities/1",
			vocab.PropertyType:   activityType,
			vocab.PropertyActor:  "https://mallory.example/@mallory",
			vocab.PropertyObject: "https://this.example/0123456789abcdef01234567",
		})

		err := DeleteAny(Context{}, activity)

		require.True(t, derp.IsForbidden(err), "%s: want Forbidden, got %v", activityType, err)
	}
}

// FuzzRequireOwnTarget verifies that the check never panics, and that an object it accepts names
// a host and shares the actor's origin
func FuzzRequireOwnTarget(f *testing.F) {

	f.Add("https://good.example/@alice", "https://good.example/notes/1")
	f.Add("https://good.example/@alice", "https://victim.example/notes/1")
	f.Add("https://good.example/@alice", "urn:uuid:1")
	f.Add("", "")
	f.Add("http://good.example", "https://good.example/x")
	f.Add("https://GOOD.example:443/@a", "https://good.example/x")

	f.Fuzz(func(t *testing.T, actor string, object string) {

		activity := streams.NewDocument(mapof.Any{
			vocab.PropertyType:   vocab.ActivityTypeDelete,
			vocab.PropertyActor:  actor,
			vocab.PropertyObject: object,
		})

		if err := requireOwnTarget(activity); err != nil {
			require.True(t, derp.IsForbidden(err))
			return
		}

		require.NotEmpty(t, object)
		require.True(t, streams.NewDocument(object).IsSameOrigin(actor))
	})
}
