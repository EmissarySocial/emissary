package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
)

// TestCleanupTransitions pins the R8 trigger table: which Rule changes require which
// retroactive passes. LABEL churn and no-op edits must enqueue nothing.
func TestCleanupTransitions(t *testing.T) {

	key := model.RuleMatchKey(model.RuleTypeActor, "https://evil.example/@spammer")
	otherKey := model.RuleMatchKey(model.RuleTypeActor, "https://noisy.example/@chatty")

	// Fresh BLOCK -> purge everything
	purgeAll, purgeNewsfeed, restore := cleanupTransitions("", "", model.RuleActionBlock, key)
	require.True(t, purgeAll)
	require.False(t, purgeNewsfeed)
	require.False(t, restore)

	// Fresh MUTE -> newsfeed purge only (D1)
	purgeAll, purgeNewsfeed, restore = cleanupTransitions("", "", model.RuleActionMute, key)
	require.False(t, purgeAll)
	require.True(t, purgeNewsfeed)
	require.False(t, restore)

	// Fresh LABEL -> nothing (labels derive at render time)
	purgeAll, purgeNewsfeed, restore = cleanupTransitions("", "", model.RuleActionLabel, key)
	require.False(t, purgeAll)
	require.False(t, purgeNewsfeed)
	require.False(t, restore)

	// BLOCK deleted -> restore only
	purgeAll, purgeNewsfeed, restore = cleanupTransitions(model.RuleActionBlock, key, "", key)
	require.False(t, purgeAll)
	require.False(t, purgeNewsfeed)
	require.True(t, restore)

	// BLOCK weakened to MUTE -> restore relationships AND purge the newsfeed
	purgeAll, purgeNewsfeed, restore = cleanupTransitions(model.RuleActionBlock, key, model.RuleActionMute, key)
	require.False(t, purgeAll)
	require.True(t, purgeNewsfeed)
	require.True(t, restore)

	// MUTE escalated to BLOCK -> full purge, nothing to restore
	purgeAll, purgeNewsfeed, restore = cleanupTransitions(model.RuleActionMute, key, model.RuleActionBlock, key)
	require.True(t, purgeAll)
	require.False(t, purgeNewsfeed)
	require.False(t, restore)

	// Unchanged BLOCK (expiry edit, etc.) -> nothing
	purgeAll, purgeNewsfeed, restore = cleanupTransitions(model.RuleActionBlock, key, model.RuleActionBlock, key)
	require.False(t, purgeAll)
	require.False(t, purgeNewsfeed)
	require.False(t, restore)

	// RULE: an edited trigger re-aims a live BLOCK -- purge for the new key, restore for the old
	purgeAll, purgeNewsfeed, restore = cleanupTransitions(model.RuleActionBlock, otherKey, model.RuleActionBlock, key)
	require.True(t, purgeAll)
	require.False(t, purgeNewsfeed)
	require.True(t, restore)

	// MUTE deleted -> nothing (nothing was paused, and purged newsfeed items stay purged -- D7)
	purgeAll, purgeNewsfeed, restore = cleanupTransitions(model.RuleActionMute, key, "", key)
	require.False(t, purgeAll)
	require.False(t, purgeNewsfeed)
	require.False(t, restore)
}
