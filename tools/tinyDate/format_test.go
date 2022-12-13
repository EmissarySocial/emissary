package tinyDate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFormat(t *testing.T) {

	{
		// Test seconds
		t1, _ := time.Parse(time.RFC3339, "2021-01-01T00:00:00Z")
		t2, _ := time.Parse(time.RFC3339, "2021-01-01T00:00:12Z")
		require.Equal(t, "12s", FormatDiff(t1, t2))
	}

	{
		// Test minutes
		t1, _ := time.Parse(time.RFC3339, "2021-01-01T00:00:00Z")
		t2, _ := time.Parse(time.RFC3339, "2021-01-01T00:12:00Z")
		require.Equal(t, "12min", FormatDiff(t1, t2))
	}

	{
		// Test hours (overflowing day)
		t1, _ := time.Parse(time.RFC3339, "2021-01-01T20:00:00Z")
		t2, _ := time.Parse(time.RFC3339, "2021-01-02T08:00:00Z")
		require.Equal(t, "12h", FormatDiff(t1, t2))
	}

	{
		// Test days
		t1, _ := time.Parse(time.RFC3339, "2021-01-01T00:00:00Z")
		t2, _ := time.Parse(time.RFC3339, "2021-01-13T23:00:00Z")
		require.Equal(t, "12d", FormatDiff(t1, t2))
	}

	{
		// Test days (overlapping month)
		t1, _ := time.Parse(time.RFC3339, "2021-01-20T00:00:00Z")
		t2, _ := time.Parse(time.RFC3339, "2021-02-01T23:00:00Z")
		require.Equal(t, "12d", FormatDiff(t1, t2))
	}

	{
		// Test months
		t1, _ := time.Parse(time.RFC3339, "2021-06-20T00:00:00Z")
		t2, _ := time.Parse(time.RFC3339, "2022-05-01T23:00:00Z")
		require.Equal(t, "11mo", FormatDiff(t1, t2))
	}
}

func TestYears(t *testing.T) {
	t1, _ := time.Parse(time.RFC3339, "2000-01-01T00:00:00Z")
	t2, _ := time.Parse(time.RFC3339, "2021-02-01T23:00:00Z")
	require.Equal(t, "21y", FormatDiff(t1, t2))
}
