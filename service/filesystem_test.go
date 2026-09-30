package service

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/tools/secretcheck"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/channel"
	"github.com/benpate/rosetta/mapof"
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

// TestFilesystemGetFSs_OneEntryPerFolder verifies that GetFSs returns one filesystem for each
// folder it can open, and nothing for a folder it cannot.
func TestFilesystemGetFSs_OneEntryPerFolder(t *testing.T) {

	filesystem := Filesystem{}
	good := mapof.String{"adapter": config.FolderAdapterFile, "location": t.TempDir()}
	bad := mapof.String{"adapter": "UNKNOWN"}

	result := filesystem.GetFSs(good, bad, good)

	require.Len(t, result, 2)
	require.NotContains(t, result, nil)
}

// TestFilesystemGetAferos_OneEntryPerFolder verifies that GetAferos returns one filesystem for
// each folder it can open, and nothing for a folder it cannot.
func TestFilesystemGetAferos_OneEntryPerFolder(t *testing.T) {

	filesystem := Filesystem{}
	good := mapof.String{"adapter": config.FolderAdapterFile, "location": t.TempDir()}
	bad := mapof.String{"adapter": "UNKNOWN"}

	result := filesystem.GetAferos(good, bad, good)

	require.Len(t, result, 2)
	require.NotContains(t, result, nil)
}

// TestFolderLabel pins what an error may say about a folder: its adapter and a location with
// every credential removed
func TestFolderLabel(t *testing.T) {

	// A Git token can ride as the password, or as the username alone
	require.Equal(t, "git https://github.com/org/repo", folderLabel(mapof.String{"adapter": "git", "location": "https://user:tok3n@github.com/org/repo"}))
	require.Equal(t, "git https://github.com/org/repo", folderLabel(mapof.String{"adapter": "git", "location": "https://tok3n@github.com/org/repo"}))

	// Locations without credentials pass through
	require.Equal(t, "embed templates", folderLabel(mapof.String{"adapter": "embed", "location": "templates"}))
	require.Equal(t, "s3 https://minio.example.com", folderLabel(mapof.String{"adapter": "s3", "location": "https://minio.example.com", "secretKey": "s3cr3t"}))

	// A location that does not parse is left out, and an empty folder names nothing
	require.Equal(t, "git", folderLabel(mapof.String{"adapter": "git", "location": "https://user:tok3n@host/%zz"}))
	require.Equal(t, " ", folderLabel(mapof.String{}))
}

// TestFilesystem_OmitsCredentialsFromErrors requires that no failure reports a folder's keys
func TestFilesystem_OmitsCredentialsFromErrors(t *testing.T) {

	// BUG-173: these errors attached the whole folder map, which holds S3 keys, or a Git
	// location that embeds a token.
	filesystem := NewFilesystem(nil)

	// A lowercase "s3" is not a supported adapter, so it fails with its keys in hand (BUG-171)
	misspelledS3 := mapof.String{"adapter": "s3", "location": "https://minio.example.com", "accessKey": "access-k3y", "secretKey": "secret-k3y", "token": "session-t0ken"}

	t.Run("GetAfero", func(t *testing.T) {
		_, err := filesystem.GetAfero(misspelledS3)
		require.Error(t, err)

		for _, secret := range []string{"access-k3y", "secret-k3y", "session-t0ken"} {
			secretcheck.RequireAbsent(t, err, secret)
		}
	})

	t.Run("GetFS", func(t *testing.T) {
		_, err := filesystem.GetFS(misspelledS3)
		require.Equal(t, "service.Filesystem.GetFS", derp.Location(err))

		for _, secret := range []string{"access-k3y", "secret-k3y", "session-t0ken"} {
			secretcheck.RequireAbsent(t, err, secret)
		}
	})

	t.Run("GitURL", func(t *testing.T) {
		_, err := filesystem.GetFS(mapof.String{"adapter": config.FolderAdapterGit, "location": "https://user:git-t0ken@host/%zz"})
		require.Equal(t, "Parsing Git URL", derp.Message(err))
		secretcheck.RequireAbsent(t, err, "git-t0ken")
	})
}
