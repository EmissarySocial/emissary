package ascache

import (
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/tools/cacheheader"
	"github.com/stretchr/testify/require"
)

// TestValue_IsExpired confirms the expiry gate that the cache-read path uses to
// decide whether a cached document may be served or must be re-fetched.
func TestValue_IsExpired(t *testing.T) {

	now := time.Now().Unix()

	// A value whose Expires is in the past is expired (must be re-fetched).
	t.Run("past expiry is expired", func(t *testing.T) {
		value := Value{Expires: now - 10}
		require.True(t, value.IsExpired())
	})

	// A value whose Expires is in the future is NOT expired (may be served).
	t.Run("future expiry is not expired", func(t *testing.T) {
		value := Value{Expires: now + 60}
		require.False(t, value.IsExpired())
	})

	// A zero Expires means "no expiry recorded" and must NOT be treated as expired,
	// otherwise every value would be re-fetched on every read.
	t.Run("zero expiry is not expired", func(t *testing.T) {
		value := Value{Expires: 0}
		require.False(t, value.IsExpired())
	})
}

// TestValue_CalcExpires confirms the precedence of the three expiry sources:
// Max-Age wins, then a valid Expires header, then the 7-day fallback.
func TestValue_CalcExpires(t *testing.T) {

	// Max-Age takes precedence over everything else.
	t.Run("max-age wins", func(t *testing.T) {
		value := NewValue()
		value.Received = 1_000
		value.HTTPHeader.Set(HeaderExpires, "Wed, 21 Oct 2099 07:28:00 GMT")
		value.calcExpires(cacheheader.Header{MaxAge: 60})
		require.Equal(t, int64(1_060), value.Expires)
	})

	// With no Max-Age, a valid Expires header is honored (regression: it used to be
	// computed and then immediately overwritten by the 7-day fallback).
	t.Run("expires header honored when no max-age", func(t *testing.T) {
		value := NewValue()
		value.HTTPHeader.Set(HeaderExpires, "Wed, 21 Oct 2099 07:28:00 GMT")
		value.calcExpires(cacheheader.Header{MaxAge: 0})

		expected, err := time.Parse(time.RFC1123, "Wed, 21 Oct 2099 07:28:00 GMT")
		require.NoError(t, err)
		require.Equal(t, expected.Unix(), value.Expires)
	})

	// A malformed Expires header falls through to the 7-day fallback.
	t.Run("malformed expires header falls back to 7 days", func(t *testing.T) {
		value := NewValue()
		value.HTTPHeader.Set(HeaderExpires, "not-a-date")
		value.calcExpires(cacheheader.Header{MaxAge: 0})

		require.False(t, value.IsExpired()) // roughly a week out
		require.Greater(t, value.Expires, time.Now().AddDate(0, 0, 6).Unix())
	})
}

// TestValue_ExpiryRevalidateParity documents the relationship the cache-read path
// relies on: by default (no Stale-While-Revalidate header) Revalidates == Expires,
// so an expired collection is both re-fetched synchronously AND flagged for reindex.
func TestValue_ExpiryRevalidateParity(t *testing.T) {

	value := NewValue()
	value.Received = time.Now().Unix()

	// Simulate the one-minute collection TTL (Max-Age: 60, no Stale-While-Revalidate).
	header := cacheheader.Header{MaxAge: 60}
	value.calcExpires(header)
	value.calcRevalidates(header)

	require.Equal(t, value.Expires, value.Revalidates,
		"without Stale-While-Revalidate, revalidation and expiry must coincide")
}
