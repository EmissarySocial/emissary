package templates

import (
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/tools/datetime"
	"github.com/stretchr/testify/require"
)

// TestInTimezone pins that inTimezone shows a time, or an Emissary DateTime, in a named timezone, and
// leaves it unchanged for an empty or unknown name.
func TestInTimezone(t *testing.T) {

	inTimezone := FuncMap(nil)["inTimezone"].(func(any, string) time.Time)
	instant := time.Date(2026, 10, 16, 1, 0, 0, 0, time.UTC)

	denver := inTimezone(instant, "America/Denver")
	require.Equal(t, "2026-10-15 19:00", denver.Format("2006-01-02 15:04"), "the same instant, on Denver's clock")
	require.True(t, denver.Equal(instant))

	fromDateTime := inTimezone(datetime.DateTime{Time: instant}, "America/Denver")
	require.Equal(t, "2026-10-15 19:00", fromDateTime.Format("2006-01-02 15:04"))

	require.Equal(t, instant, inTimezone(instant, ""))
	require.Equal(t, instant, inTimezone(instant, "Not/AZone"))
}
