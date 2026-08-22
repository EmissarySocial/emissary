package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestUpdateChannel_DeliversToAWaitingReader covers the ordinary path.
func TestUpdateChannel_DeliversToAWaitingReader(t *testing.T) {

	channel := newUpdateChannel()

	config := NewConfig()
	config.AdminEmail = "first@example.com"

	channel.notify(config)

	require.Equal(t, "first@example.com", (<-channel.subscribe()).AdminEmail)
}

// TestUpdateChannel_NeverBlocks is the important one. A storage engine's watcher goroutine calls
// notify while it is the only thing watching for changes: if notify can park, the watcher stops
// watching. For the MongoDB change stream that is fatal rather than merely slow -- a long enough
// stall pushes the resume token out of the oplog window and kills the stream, which is exactly how
// a node used to end up frozen on a stale configuration until someone rebooted it.
func TestUpdateChannel_NeverBlocks(t *testing.T) {

	channel := newUpdateChannel()
	finished := make(chan struct{})

	go func() {
		defer close(finished)

		// Nobody is reading the channel, and its buffer holds exactly one entry
		for index := range 1000 {
			config := NewConfig()
			config.HTTPPort = index
			channel.notify(config)
		}
	}()

	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("notify blocked with no reader draining the channel")
	}
}

// TestUpdateChannel_LatestWins pins the discard policy: a configuration that is still queued is
// replaced rather than delivered. Configuration is full state, so an older version is never worth
// applying -- and applying it would be actively wrong, since it would land after the newer one.
func TestUpdateChannel_LatestWins(t *testing.T) {

	channel := newUpdateChannel()

	for index := range 10 {
		config := NewConfig()
		config.HTTPPort = index
		channel.notify(config)
	}

	require.Equal(t, 9, (<-channel.subscribe()).HTTPPort)
	require.Zero(t, len(channel), "only the newest configuration should be queued")
}

// TestUpdateChannel_SurvivesAConcurrentConsumer pins that the discard loop terminates when a
// reader is draining at the same time. The loop is only safe because there is exactly one
// publisher; this exercises the race between publisher and consumer that it has to tolerate.
func TestUpdateChannel_SurvivesAConcurrentConsumer(t *testing.T) {

	channel := newUpdateChannel()
	published := make(chan struct{})
	consumed := make(chan struct{})

	// Drain until the publisher is finished. The consumer takes NO position on how many values it
	// should see: latest-wins means the count is legitimately nondeterministic, so a consumer
	// waiting for a fixed number would hang whenever the publisher outran it.
	go func() {
		defer close(consumed)

		for {
			select {
			case <-channel.subscribe():
			case <-published:
				return
			}
		}
	}()

	go func() {
		defer close(published)

		for index := range 5000 {
			config := NewConfig()
			config.HTTPPort = index
			channel.notify(config)
		}
	}()

	select {

	case <-consumed:

	case <-time.After(10 * time.Second):
		t.Fatal("notify deadlocked against a concurrent consumer")
	}
}
