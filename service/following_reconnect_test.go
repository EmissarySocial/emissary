package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
)

// TestReconnectsOnSave pins which statuses a form save restarts. The rule is "if it is showing
// the user a problem, saving it retries" (D5) -- FAILURE included, because that is the state an
// owner is most likely to open and re-save trying to fix.
func TestReconnectsOnSave(t *testing.T) {

	// The first connection, and every retry
	require.True(t, reconnectsOnSave(model.FollowingStatusNew))
	require.True(t, reconnectsOnSave(model.FollowingStatusFailure))
	require.True(t, reconnectsOnSave(model.FollowingStatusPaused))
	require.True(t, reconnectsOnSave(model.FollowingStatusGone))

	// RULE: A healthy record must not fire a fresh Follow on every folder edit
	require.False(t, reconnectsOnSave(model.FollowingStatusSuccess))

	// Already connecting
	require.False(t, reconnectsOnSave(model.FollowingStatusLoading))

	// Owned by BUG-156; this plan takes no position on it
	require.False(t, reconnectsOnSave(model.FollowingStatusImportPending))

	// RULE: R11 -- re-following a blocked actor is an explicit decision through Follow(),
	// never a side effect of a save. Save refuses BLOCKED before it gets here anyway.
	require.False(t, reconnectsOnSave(model.FollowingStatusBlocked))

	// Unknown and empty statuses are left alone rather than reconnected
	require.False(t, reconnectsOnSave(""))
	require.False(t, reconnectsOnSave("WAT"))

	// Every status has a decision, and exactly four say yes
	yes := 0
	for _, status := range everyFollowingStatus() {
		if reconnectsOnSave(status) {
			yes++
		}
	}
	require.Equal(t, 4, yes)
}
