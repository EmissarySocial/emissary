package server

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/benpate/floodgate"
	"github.com/benpate/hannibal/delivery"
	"github.com/stretchr/testify/require"
)

// TestHostOf verifies that a host is named in lower case with no port, and that a value with no
// host names nothing
func TestHostOf(t *testing.T) {
	require.Equal(t, "remote.example", hostOf("https://remote.example/@alice"))
	require.Equal(t, "remote.example", hostOf("HTTPS://Remote.EXAMPLE/@alice"))
	require.Equal(t, "remote.example", hostOf("https://remote.example:8443/@alice"))
	require.Equal(t, "remote.example", hostOf("https://user:secret@remote.example/@alice"))
	require.Empty(t, hostOf(""))
	require.Empty(t, hostOf("not a url"))
	require.Empty(t, hostOf("mailto:alice@remote.example"))
	require.Empty(t, hostOf("://bad"))
}

// TestHostOf_MatchesDeliveryKey verifies that an inbound request revives the same record that
// hannibal's delivery pipeline reads, however the URL spells the host
func TestHostOf_MatchesDeliveryKey(t *testing.T) {

	for _, value := range []string{
		"https://remote.example/@alice",
		"https://REMOTE.example:443/@alice",
		"https://remote.example:8443/inbox",
		"http://Remote.Example/users/alice#main-key",
		"mailto:alice@remote.example",
		"not a url",
		"",
	} {
		require.Equal(t, delivery.HostOf(value), hostOf(value), value)
	}
}

// blocklistOf returns a Blocklist of the provided hosts
func blocklistOf(hosts ...string) floodgate.Blocklist {

	result := make(floodgate.Blocklist, len(hosts))

	for _, host := range hosts {
		result[host] = struct{}{}
	}

	return result
}

// TestBlockedHostCache_ReloadDoesNotBlockReaders verifies that a caller answers from the old copy
// while another caller's reload is still waiting on the database (BUG-381)
func TestBlockedHostCache_ReloadDoesNotBlockReaders(t *testing.T) {

	cache := blockedHostCache{hosts: blocklistOf("old.example")}
	started := make(chan struct{})
	release := make(chan struct{})
	reloaded := make(chan bool)

	go func() {
		isBlocked, _ := cache.contains("new.example", func(context.Context) (floodgate.Blocklist, error) {
			close(started)
			<-release
			return blocklistOf("new.example"), nil
		})
		reloaded <- isBlocked
	}()

	<-started
	answered := make(chan bool)

	go func() {
		isBlocked, _ := cache.contains("old.example", func(context.Context) (floodgate.Blocklist, error) {
			t.Error("a second caller must not reload")
			return nil, nil
		})
		answered <- isBlocked
	}()

	select {
	case isBlocked := <-answered:
		require.True(t, isBlocked, "the reader answers from the old copy")
	case <-time.After(5 * time.Second):
		t.Fatal("a reader waited for another caller's reload")
	}

	close(release)
	require.True(t, <-reloaded, "the reloading caller answers from the new copy")
}

// TestBlockedHostCache_OneReloadAtATime verifies that many callers who find the copy expired
// make one database read between them
func TestBlockedHostCache_OneReloadAtATime(t *testing.T) {

	cache := blockedHostCache{}
	var loads atomic.Int32
	var group sync.WaitGroup

	for range 50 {
		group.Go(func() {
			_, _ = cache.contains("remote.example", func(context.Context) (floodgate.Blocklist, error) {
				loads.Add(1)
				return blocklistOf("remote.example"), nil
			})
		})
	}

	group.Wait()
	require.Equal(t, int32(1), loads.Load())
}

// TestBlockedHostCache_FailedReloadKeepsOldCopy verifies that a failed reload answers from the
// old copy, and is not tried again inside the same interval
func TestBlockedHostCache_FailedReloadKeepsOldCopy(t *testing.T) {

	cache := blockedHostCache{hosts: blocklistOf("old.example")}
	loads := 0

	fail := func(context.Context) (floodgate.Blocklist, error) {
		loads++
		return nil, errors.New("database is down")
	}

	isBlocked, err := cache.contains("old.example", fail)
	require.Error(t, err)
	require.True(t, isBlocked)

	isBlocked, err = cache.contains("old.example", fail)
	require.NoError(t, err)
	require.True(t, isBlocked)
	require.Equal(t, 1, loads)
}

// TestBlockedHostCache_RemoveDoesNotForceReload verifies that a revived host stops reading as
// blocked at once, and that removing it leaves the copy fresh
func TestBlockedHostCache_RemoveDoesNotForceReload(t *testing.T) {

	cache := blockedHostCache{}
	loads := 0

	load := func(context.Context) (floodgate.Blocklist, error) {
		loads++
		return blocklistOf("paused.example", "other.example"), nil
	}

	isBlocked, err := cache.contains("paused.example", load)
	require.NoError(t, err)
	require.True(t, isBlocked)

	for range 10 {
		cache.remove("paused.example")
		isBlocked, _ = cache.contains("paused.example", load)
		require.False(t, isBlocked)
	}

	isBlocked, _ = cache.contains("other.example", load)
	require.True(t, isBlocked, "the other hosts are kept")
	require.Equal(t, 1, loads)
}

// TestBlockedHostCache_RemoveFromEmptyCopy verifies that removing a host before the first reload is harmless
func TestBlockedHostCache_RemoveFromEmptyCopy(t *testing.T) {

	cache := blockedHostCache{}
	cache.remove("remote.example")
	require.Empty(t, cache.hosts)
}
