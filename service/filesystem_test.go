package service

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/benpate/rosetta/channel"
	"github.com/stretchr/testify/require"
)

// TestFilesystemWatchOS_StopsWhileSending is BUG-180 task G: a file event that arrives as the
// template watcher stops must not strand the filesystem watcher.  Nothing reads "changes" after
// the stop, so a watcher blocked on that send has to give up when "done" closes.  It used to wait
// forever, and before that the channel was closed under it, which panicked the process.
func TestFilesystemWatchOS_StopsWhileSending(t *testing.T) {

	directory := t.TempDir()
	baseline := runtime.NumGoroutine()

	// An unbuffered channel that nobody reads, like the one a stopped template watcher leaves
	changes := make(chan bool)
	done := make(chan channel.Done)

	filesystem := Filesystem{}
	require.NoError(t, filesystem.watchOS(directory, changes, done))

	// A file event leaves the watcher blocked on its send
	require.NoError(t, os.WriteFile(filepath.Join(directory, "template.json"), []byte("{}"), 0o600))
	time.Sleep(200 * time.Millisecond)

	close(done)

	// Polled by hand, because require.Eventually runs its condition in a goroutine of its own
	deadline := time.Now().Add(5 * time.Second)

	for runtime.NumGoroutine() > baseline {
		require.True(t, time.Now().Before(deadline), "the filesystem watcher did not stop while a change was pending")
		time.Sleep(20 * time.Millisecond)
	}
}
