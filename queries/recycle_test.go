package queries

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

// bounds pulls the two range operators out of the filter that recycleFilter builds.
func bounds(t *testing.T, filter bson.M) bson.M {

	t.Helper()

	deleteDate, isMap := filter["deleteDate"].(bson.M)
	require.True(t, isMap, "filter must constrain deleteDate")

	return deleteDate
}

// TestRecycleFilter_ExcludesLiveRecords pins the single most dangerous property of this query.
// deleteDate is ZERO on every live record, so without the `$gt: 0` bound, `$lt: cutoff` would match
// every live record and Recycle would purge the entire database.
func TestRecycleFilter_ExcludesLiveRecords(t *testing.T) {

	deleteDate := bounds(t, recycleFilter(time.Now()))

	require.Equal(t, 0, deleteDate["$gt"], "`$gt: 0` guards LIVE records (deleteDate == 0) -- never remove it")

	// A live record's deleteDate (0) must fall outside the filter's lower bound.
	require.False(t, 0 > deleteDate["$gt"].(int), "deleteDate of 0 must never satisfy the lower bound")
}

// TestRecycleFilter_CutoffIsMilliseconds confirms the cutoff matches journal.SetDeleted's units.
// A seconds-based cutoff would be ~1000x too small, silently matching nothing.
func TestRecycleFilter_CutoffIsMilliseconds(t *testing.T) {

	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	deleteDate := bounds(t, recycleFilter(now))

	require.Equal(t, now.Add(-30*24*time.Hour).UnixMilli(), deleteDate["$lt"])

	// Guard the units directly: a millisecond epoch for any modern date exceeds 1e12, while the
	// equivalent seconds epoch is ~1e9.
	require.Greater(t, deleteDate["$lt"].(int64), int64(1_000_000_000_000))
}

// TestRecycleFilter_Window confirms which records fall inside the 30-day retention window.
func TestRecycleFilter_Window(t *testing.T) {

	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	cutoff := bounds(t, recycleFilter(now))["$lt"].(int64)

	// Deleted 31 days ago: expired, and therefore purgable.
	require.Less(t, now.Add(-31*24*time.Hour).UnixMilli(), cutoff)

	// Deleted 29 days ago: still inside the retention window, and must be retained.
	require.Greater(t, now.Add(-29*24*time.Hour).UnixMilli(), cutoff)

	// Deleted one second ago: must be retained.
	require.Greater(t, now.Add(-time.Second).UnixMilli(), cutoff)
}
