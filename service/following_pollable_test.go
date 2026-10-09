package service

import (
	"fmt"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
)

// matchFollowing evaluates pollableCriteria against one in-memory Following, so the polling
// rule can be pinned without a database. It PANICS on a predicate it does not understand:
// a change that adds a field or operator to the criteria must fail this test loudly, not pass
// it by accident.
func matchFollowing(status string, nextPoll int64) exp.MatcherFunc {

	return func(predicate exp.Predicate) bool {

		switch predicate.Field {

		case "status":
			switch predicate.Operator {
			case exp.OperatorEqual:
				return status == predicate.Value
			case exp.OperatorNotEqual:
				return status != predicate.Value
			}

		case "nextPoll":
			if predicate.Operator == exp.OperatorLessThan {
				return nextPoll < predicate.Value.(int64)
			}
		}

		panic(fmt.Sprintf("pollableCriteria uses a predicate this test does not model: %s %s %v", predicate.Field, predicate.Operator, predicate.Value))
	}
}

// everyFollowingStatus lists every status a Following record can hold.
func everyFollowingStatus() []string {
	return []string{
		model.FollowingStatusNew,
		model.FollowingStatusLoading,
		model.FollowingStatusImportPending,
		model.FollowingStatusSuccess,
		model.FollowingStatusFailure,
		model.FollowingStatusPaused,
		model.FollowingStatusGone,
		model.FollowingStatusBlocked,
	}
}

// TestPollableCriteria pins the one query that decides which Followings are contacted. Its
// status exclusions are the entire enforcement of "a blocked actor is never contacted" (R8),
// and nothing pinned them before.
func TestPollableCriteria(t *testing.T) {

	const now = int64(1_700_000_000)
	criteria := pollableCriteria(now)

	due := now - 1
	notDue := now + 1

	// RULE: BLOCKED is never polled, however overdue it is. This is R8.
	require.False(t, criteria.Match(matchFollowing(model.FollowingStatusBlocked, due)))
	require.False(t, criteria.Match(matchFollowing(model.FollowingStatusBlocked, 0)))

	// RULE: GONE is never polled. The remote server said the account is deleted (D4).
	require.False(t, criteria.Match(matchFollowing(model.FollowingStatusGone, due)))
	require.False(t, criteria.Match(matchFollowing(model.FollowingStatusGone, 0)))

	// RULE: PAUSED IS polled once its 90-day NextPoll arrives. Excluding it here would turn the
	// backoff into a graveyard, and the quarterly retry is the only automatic way back.
	require.True(t, criteria.Match(matchFollowing(model.FollowingStatusPaused, due)))
	require.False(t, criteria.Match(matchFollowing(model.FollowingStatusPaused, notDue)))

	// Every other status is polled when due, and only when due
	for _, status := range everyFollowingStatus() {

		if status == model.FollowingStatusBlocked || status == model.FollowingStatusGone {
			continue
		}

		require.True(t, criteria.Match(matchFollowing(status, due)), "%s must be polled when due", status)
		require.False(t, criteria.Match(matchFollowing(status, notDue)), "%s must not be polled before it is due", status)
	}

	// The date test is strict: a record due exactly now waits for the next sweep
	require.False(t, criteria.Match(matchFollowing(model.FollowingStatusSuccess, now)))

	// Exactly two statuses are excluded -- no more, no fewer
	excluded := 0
	for _, status := range everyFollowingStatus() {
		if !criteria.Match(matchFollowing(status, due)) {
			excluded++
		}
	}
	require.Equal(t, 2, excluded, "only BLOCKED and GONE are excluded from polling")
}
