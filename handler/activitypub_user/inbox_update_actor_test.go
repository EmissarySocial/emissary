package activitypub_user

import (
	"testing"

	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestInbox_UpdateActor_RejectsMismatchedActor pins the self-update rule: an actor may only
// Update its own profile. The zero Context proves the guard runs before any factory access —
// a fall-through to the cache purge would nil-panic the test.
func TestInbox_UpdateActor_RejectsMismatchedActor(t *testing.T) {

	activity := streams.NewDocument(mapof.Any{
		vocab.PropertyType:  vocab.ActivityTypeUpdate,
		vocab.PropertyActor: "https://mallory.example/@mallory",
		vocab.PropertyObject: mapof.Any{
			vocab.PropertyID:   "https://victim.example/@alice",
			vocab.PropertyType: vocab.ActorTypePerson,
		},
	})

	require.Error(t, inbox_UpdateActor(Context{}, activity))
}
