package consumer

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestPollFollowingSignature pins the dedupe key that keeps a four-hour sweep from queueing a
// second poll for a Following whose previous task is still working through turbine's retry
// chain. The chain can run ~4h15m, which outlasts the sweep interval (BUG-148).
func TestPollFollowingSignature(t *testing.T) {

	first := primitive.NewObjectID()
	second := primitive.NewObjectID()

	// The same Following always produces the same signature, so a duplicate enqueue collapses
	require.Equal(t, pollFollowingSignature(first), pollFollowingSignature(first))

	// RULE: Two different Followings must NEVER share a signature. A signature keyed on
	// anything coarser (the UserID, say) would collapse every follow of one User into a
	// single task and silently leave the rest unpolled.
	require.NotEqual(t, pollFollowingSignature(first), pollFollowingSignature(second))

	// The signature is namespaced by task name, so it cannot collide with another task type
	// that happens to key on the same ObjectID.
	require.Equal(t, "PollFollowing-Record:"+first.Hex(), pollFollowingSignature(first))

	// A zero ObjectID is still a distinct, non-empty signature rather than a bare prefix that
	// every uninitialized record would share.
	zero := pollFollowingSignature(primitive.NilObjectID)
	require.NotEmpty(t, zero)
	require.NotEqual(t, "PollFollowing-Record:", zero)

	// Signatures are unique across a batch, which is what makes per-record dedupe work at all
	seen := make(map[string]bool)
	for range 500 {
		signature := pollFollowingSignature(primitive.NewObjectID())
		require.False(t, seen[signature], "duplicate signature %s", signature)
		seen[signature] = true
	}
}
