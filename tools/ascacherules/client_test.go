package ascacherules

import (
	"testing"

	"github.com/EmissarySocial/emissary/tools/cacheheader"
	"github.com/stretchr/testify/require"
)

// TestActorMaxAge pins the cache window for ActivityPub Actor documents.  This is a security
// parameter, not a performance knob: the window in which a stale Actor is served is also the window
// in which a REVOKED key keeps being accepted (BUG-22 D2).
func TestActorMaxAge(t *testing.T) {

	// An origin that says nothing gets our default.  Silence and `max-age=0` parse identically, so
	// this branch also covers the (many) peers that send no Cache-Control at all.
	t.Run("silence uses the default", func(t *testing.T) {
		require.Equal(t, int64(12*hour), actorMaxAge(cacheheader.Header{}))
	})

	// An origin asking to stay fresh is HONORED. The old rule promoted Mastodon's 3-minute max-age
	// to a full day, overriding a peer that was actively asking to be re-fetched.
	t.Run("short max-age is honored", func(t *testing.T) {
		require.Equal(t, int64(180), actorMaxAge(cacheheader.Header{MaxAge: 180}))
	})

	// A shared cache prefers s-maxage when the origin states both.
	t.Run("s-maxage wins over max-age", func(t *testing.T) {
		require.Equal(t, int64(300), actorMaxAge(cacheheader.Header{SMaxAge: 300, MaxAge: 900}))
	})

	// The ceiling is the change that matters most: a month down to a day.
	t.Run("ceiling is one day", func(t *testing.T) {
		require.Equal(t, int64(day), actorMaxAge(cacheheader.Header{MaxAge: month}))
	})

	// A floor keeps a peer from making its Actor effectively uncacheable one request at a time.
	t.Run("floor is one minute", func(t *testing.T) {
		require.Equal(t, int64(minute), actorMaxAge(cacheheader.Header{MaxAge: 5}))
	})

	// The default must sit under the ceiling, or the ceiling would never bind.
	t.Run("default is below the ceiling", func(t *testing.T) {
		require.Less(t, actorMaxAge(cacheheader.Header{}), int64(day))
	})
}
