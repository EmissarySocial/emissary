package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestStream_SetDateTime pins that a form's date-time string sets startDate and endDate, and that an
// empty string clears one, so an optional end date can be left blank (qwerty-event's form).
func TestStream_SetDateTime(t *testing.T) {

	stream := NewStream()

	require.True(t, stream.SetString("startDate", "2026-10-15T19:00"))
	require.Equal(t, time.Date(2026, 10, 15, 19, 0, 0, 0, time.UTC), stream.StartDate.Time)

	require.True(t, stream.SetString("endDate", "2026-10-15T21:30"))
	require.False(t, stream.EndDate.IsZero())

	require.True(t, stream.SetString("endDate", ""))
	require.True(t, stream.EndDate.IsZero(), "an empty value clears the date")

	require.False(t, stream.SetString("startDate", "not a date"))
	require.Equal(t, time.Date(2026, 10, 15, 19, 0, 0, 0, time.UTC), stream.StartDate.Time, "a bad value leaves the date alone")
}
