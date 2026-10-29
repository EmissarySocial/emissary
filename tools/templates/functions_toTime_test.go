package templates

import (
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/tools/datetime"
	"github.com/stretchr/testify/require"
)

// TestToTime pins that toTime reads an Emissary DateTime, a date-only string, and nonsense, for the calendar's ?startDate=
func TestToTime(t *testing.T) {

	toTime := FuncMap(nil)["toTime"].(func(any) time.Time)
	instant := time.Date(2026, 10, 20, 19, 0, 0, 0, time.UTC)

	require.Equal(t, instant, toTime(datetime.DateTime{Time: instant}))
	require.Equal(t, instant, toTime(&datetime.DateTime{Time: instant}))
	require.Equal(t, time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC), toTime("2026-10-20"))
	require.True(t, toTime("not a date").IsZero())
	require.True(t, toTime((*datetime.DateTime)(nil)).IsZero())
}
