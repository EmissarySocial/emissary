package queries

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestPlanNotificationTrim_UnderCap confirms that a User at or below the cap is never trimmed.
func TestPlanNotificationTrim_UnderCap(t *testing.T) {

	// Below the cap.
	require.Equal(t, notificationTrimPlan{}, planNotificationTrim(1500, 1000, 2000))

	// Exactly at the cap: still nothing (surplus is zero).
	require.Equal(t, notificationTrimPlan{}, planNotificationTrim(2000, 500, 2000))
}

// TestPlanNotificationTrim_ReadRowsAbsorbSurplus confirms the read-first rule: when there are enough
// READ rows to cover the whole surplus, only read rows are deleted and every unread row is kept.
func TestPlanNotificationTrim_ReadRowsAbsorbSurplus(t *testing.T) {

	// 500 over cap, plenty of read rows: delete 500 read, 0 unread.
	require.Equal(t, notificationTrimPlan{Read: 500, Unread: 0}, planNotificationTrim(2500, 2400, 2000))

	// Surplus exactly equals the read count: delete every read row, still no unread.
	require.Equal(t, notificationTrimPlan{Read: 500, Unread: 0}, planNotificationTrim(2500, 500, 2000))
}

// TestPlanNotificationTrim_SpillsIntoUnread confirms that when read rows cannot cover the surplus,
// every read row is deleted and the remainder comes from the oldest unread rows.
func TestPlanNotificationTrim_SpillsIntoUnread(t *testing.T) {

	// 500 over cap, only 200 read rows: delete all 200 read, plus 300 unread.
	require.Equal(t, notificationTrimPlan{Read: 200, Unread: 300}, planNotificationTrim(2500, 200, 2000))

	// No read rows at all (pathological all-unread flood): the entire surplus comes from unread.
	require.Equal(t, notificationTrimPlan{Read: 0, Unread: 500}, planNotificationTrim(2500, 0, 2000))
}

// TestPlanNotificationTrim_NeverExceedsAvailable confirms the plan never asks to delete more read
// rows than exist, and that Read + Unread always equals the surplus when over the cap.
func TestPlanNotificationTrim_NeverExceedsAvailable(t *testing.T) {

	cases := []struct {
		count     int64
		readCount int64
		capacity  int64
	}{
		{2001, 0, 2000},
		{2001, 2001, 2000},
		{10000, 1, 2000},
		{10000, 9999, 2000},
		{3000, 1000, 2000},
	}

	for _, testCase := range cases {

		plan := planNotificationTrim(testCase.count, testCase.readCount, testCase.capacity)
		surplus := testCase.count - testCase.capacity

		require.LessOrEqual(t, plan.Read, testCase.readCount, "never delete more read rows than exist")
		require.GreaterOrEqual(t, plan.Read, int64(0), "read count is never negative")
		require.GreaterOrEqual(t, plan.Unread, int64(0), "unread count is never negative")
		require.Equal(t, surplus, plan.Read+plan.Unread, "total deleted equals the surplus over the cap")
	}
}
