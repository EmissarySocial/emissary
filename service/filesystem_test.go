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

// TestFilesystemWatchOS_StopsWhileSending verifies that a filesystem watcher blocked on
// sending a change gives up when "done" closes
func TestFilesystemWatchOS_StopsWhileSending(t *testing.T) {

	// BUG-180 task G: nothing reads "changes" once the template watcher stops.  This send
	// used to wait forever, and before that the channel was closed under it, which panicked.
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
