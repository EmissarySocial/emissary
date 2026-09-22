package service

import (
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
)

// newUnresponsiveTestFollowing returns a Following with the error count and last-success time that
// isUnresponsive reads, and nothing else.
func newUnresponsiveTestFollowing(errorCount int, lastPolled int64) *model.Following {
	following := model.NewFollowing()
	following.ErrorCount = errorCount
	following.LastPolled = lastPolled
	return &following
}

// TestIsUnresponsive pins the rule that backs a source off to PAUSED. BOTH conditions are required: elapsed time
// alone would abandon an ActivityPub follow after ONE failed 30-day poll, and the error count
// alone is five days for a POLL follow but 150 days for an ActivityPub one.
func TestIsUnresponsive(t *testing.T) {

	const day = int64(24 * 60 * 60)
	now := time.Now().Unix()

	// The qualifying case: long enough AND often enough
	require.True(t, isUnresponsive(newUnresponsiveTestFollowing(5, now-(31*day)), now))
	require.True(t, isUnresponsive(newUnresponsiveTestFollowing(500, now-(365*day)), now))

	// RULE: Time alone is not enough. An ActivityPub follow polls every 30 days, so one failed
	// poll already clears the time bar -- abandoning on that alone would kill it after ONE miss.
	require.False(t, isUnresponsive(newUnresponsiveTestFollowing(1, now-(31*day)), now))
	require.False(t, isUnresponsive(newUnresponsiveTestFollowing(4, now-(365*day)), now))

	// RULE: Errors alone are not enough. A POLL follow reaches five errors in five days.
	require.False(t, isUnresponsive(newUnresponsiveTestFollowing(5, now-(5*day)), now))
	require.False(t, isUnresponsive(newUnresponsiveTestFollowing(100, now-(29*day)), now))

	// The boundary is exclusive: exactly 30 days is not yet unresponsive
	require.False(t, isUnresponsive(newUnresponsiveTestFollowing(5, now-(30*day)), now))
	require.True(t, isUnresponsive(newUnresponsiveTestFollowing(5, now-(30*day)-1), now))

	// A record that has never succeeded is measured from its CreateDate instead.
	// RULE: Journal dates are MILLISECONDS and LastPolled is seconds -- mixing them would make
	// every record look about 50,000 years old, and abandon it on its fifth error.
	neverPolled := newUnresponsiveTestFollowing(5, 0)
	neverPolled.CreateDate = (now - (31 * day)) * 1000
	require.True(t, isUnresponsive(neverPolled, now))

	recentlyCreated := newUnresponsiveTestFollowing(5, 0)
	recentlyCreated.CreateDate = (now - (5 * day)) * 1000
	require.False(t, isUnresponsive(recentlyCreated, now))

	// RULE: With no usable baseline at all we cannot say how long this has been broken, so we
	// do not guess. A zero here would otherwise read as 1970 and abandon the record instantly.
	noBaseline := newUnresponsiveTestFollowing(1000, 0)
	noBaseline.CreateDate = 0
	require.False(t, isUnresponsive(noBaseline, now))

	// A future LastPolled (clock skew) is never unresponsive
	require.False(t, isUnresponsive(newUnresponsiveTestFollowing(1000, now+(365*day)), now))

	// A brand new Following is never unresponsive
	fresh := model.NewFollowing()
	require.False(t, isUnresponsive(&fresh, now))
}

// TestPausedRecheckSeconds pins the recovery interval. PAUSED must never be permanent: the
// quarterly retry is what brings a source back when the cause was on OUR side, which is exactly
// what happened in BUG-148's Defect B.
func TestPausedRecheckSeconds(t *testing.T) {

	require.Equal(t, int64(90*24*60*60), pausedRecheckSeconds)

	// It must comfortably exceed the longest ordinary cadence (30 days for ActivityPub),
	// or a paused record would be polled more often than a healthy one.
	require.Greater(t, pausedRecheckSeconds, int64(30*24*60*60))

	// And it must be finite, which is the whole point
	require.Positive(t, pausedRecheckSeconds)
}
