package service

import (
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"

	"github.com/stretchr/testify/require"
)

// TestFollowingBackoff pins the exponential ladder that keeps a permanently-broken Following
// from being re-polled every sweep. The original `2 ^ exponent` was Go's XOR operator rather
// than a power, which produced 3, 0, 1, 6, 7, 4, 5, 10 minutes -- note the ZERO at two
// consecutive failures, which re-polled the record immediately (BUG-148).
func TestFollowingBackoff(t *testing.T) {

	// The documented ladder: 1m, 2m, 4m ... 256m, indexed by consecutive error count.
	expected := map[int]time.Duration{
		1: 1 * time.Minute,
		2: 2 * time.Minute,
		3: 4 * time.Minute,
		4: 8 * time.Minute,
		5: 16 * time.Minute,
		6: 32 * time.Minute,
		7: 64 * time.Minute,
		8: 128 * time.Minute,
		9: 256 * time.Minute,
	}

	for errorCount, want := range expected {
		require.Equal(t, want, followingBackoff(errorCount), "errorCount %d", errorCount)
	}

	// Every failure past the cap holds at 256 minutes, which is just over the four-hour
	// PollFollowing-Index sweep -- so a permanent failure is skipped by most sweeps.
	require.Equal(t, 256*time.Minute, followingBackoff(10))
	require.Equal(t, 256*time.Minute, followingBackoff(100))
	require.Equal(t, 256*time.Minute, followingBackoff(1_000_000))

	// RULE: No backoff is ever zero. A zero makes the record immediately pollable again,
	// which is the exact retry loop this function exists to prevent.
	for errorCount := -1000; errorCount < 1000; errorCount++ {
		require.Positive(t, followingBackoff(errorCount), "errorCount %d", errorCount)
	}

	// A shift by a negative count panics in Go, so a non-positive ErrorCount (never produced
	// by SetStatusFailure, but reachable from a hand-edited record) must clamp rather than crash.
	// Reaching these assertions at all is the proof that it did not panic.
	require.Equal(t, 1*time.Minute, followingBackoff(0))
	require.Equal(t, 1*time.Minute, followingBackoff(-1))

	// The ladder never decreases, at any input
	previous := followingBackoff(-5)
	for errorCount := -5; errorCount < 50; errorCount++ {
		current := followingBackoff(errorCount)
		require.GreaterOrEqual(t, current, previous, "errorCount %d went backwards", errorCount)
		previous = current
	}
}

// TestNextPollDate pins the cadence half of the polling schedule. This is the date that both
// poll outcomes now write, and it is the ONLY thing standing between a Following and being
// re-selected by every four-hour PollFollowing-Index sweep (BUG-148).
func TestNextPollDate(t *testing.T) {

	const hour = int64(60 * 60)
	lastPolled := int64(1_700_000_000)

	// The ordinary case: one PollDuration (in hours) after the attempt
	require.Equal(t, lastPolled+(24*hour), nextPollDate(&model.Following{PollDuration: 24}, lastPolled))
	require.Equal(t, lastPolled+(8*hour), nextPollDate(&model.Following{PollDuration: 8}, lastPolled))
	require.Equal(t, lastPolled+hour, nextPollDate(&model.Following{PollDuration: 1}, lastPolled))

	// The ActivityPub cadence (30 days) does not overflow
	activityPub := &model.Following{PollDuration: 24 * 7 * 30}
	require.Equal(t, lastPolled+(5040*hour), nextPollDate(activityPub, lastPolled))
	require.Greater(t, nextPollDate(activityPub, lastPolled), lastPolled)

	// RULE: A missing or corrupt PollDuration falls back to the 24-hour default. A zero here
	// would schedule the next poll in the PAST, putting the record in every single sweep.
	for _, pollDuration := range []int{0, -1, -24, -1_000_000} {
		following := &model.Following{PollDuration: pollDuration}
		require.Equal(t, lastPolled+(24*hour), nextPollDate(following, lastPolled), "pollDuration %d", pollDuration)
	}

	// RULE: The result is ALWAYS in the future relative to the attempt, for every input.
	for _, pollDuration := range []int{-100, -1, 0, 1, 24, 168, 5040, 100_000} {
		following := &model.Following{PollDuration: pollDuration}
		require.Greater(t, nextPollDate(following, lastPolled), lastPolled, "pollDuration %d", pollDuration)
	}

	// A zero "lastPolled" (a Following that has never been polled) is handled, not special-cased
	require.Equal(t, 24*hour, nextPollDate(&model.Following{PollDuration: 24}, 0))
}
