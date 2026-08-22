package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

// These tests cover the supervised half of FileStorage: the filesystem watcher that tells a
// running process its configuration file changed.  The behavior that matters is not "does one
// write land on disk" -- it is "does the process still see changes after the watch dies," which
// on the file engine happens quietly on every atomic-rename save (vim and most editors).

// newFileTestStorage builds a FileStorage around a real config file in a temp directory, without
// calling NewFileStorage (which exits the process on failure and is therefore untestable).
func newFileTestStorage(t *testing.T) FileStorage {

	t.Helper()

	path := filepath.Join(t.TempDir(), "config.json")

	storage := FileStorage{
		source:        ConfigSourceCommandLine,
		location:      path,
		updateChannel: newUpdateChannel(),
		closeChannel:  make(chan struct{}),
		closeOnce:     &sync.Once{},
	}

	writeFileConfig(t, storage, "initial@example.com")

	t.Cleanup(storage.Close)

	return storage
}

// writeFileConfig writes a minimal valid configuration to the storage's file, in place.  It
// reads the current revision first, so repeated calls satisfy Write's compare-and-swap.
func writeFileConfig(t *testing.T, storage FileStorage, adminEmail string) {

	t.Helper()

	config := DefaultConfig()
	config.AdminEmail = adminEmail

	if current, err := storage.load(); err == nil {
		config.Revision = current.Revision
	}

	_, err := storage.Write(config)
	require.NoError(t, err)
}

// renameFileConfig writes the configuration the way editors do: a temp file, atomically renamed
// over the original.  This REPLACES the inode the watcher is attached to, which is the quiet
// death the supervisor exists to survive.
func renameFileConfig(t *testing.T, storage FileStorage, adminEmail string) {

	t.Helper()

	config := DefaultConfig()
	config.AdminEmail = adminEmail

	data, err := json.Marshal(config)
	require.NoError(t, err)

	temporary := storage.location + ".tmp"
	require.NoError(t, os.WriteFile(temporary, data, 0600))
	require.NoError(t, os.Rename(temporary, storage.location))
}

// awaitFileConfig drains published configurations until one satisfies `match`, and fails if none
// does before the timeout.  Same discipline as the MongoDB tests: wait for the configuration you
// EXPECT, never assert on whichever one arrives first -- the update channel keeps only the
// newest entry, and a reopened watcher re-reads the file, so duplicates and gaps are both normal.
func awaitFileConfig(t *testing.T, storage FileStorage, timeout time.Duration, match func(Config) bool) Config {

	t.Helper()

	for deadline := time.Now().Add(timeout); time.Now().Before(deadline); {

		select {

		case result := <-storage.Subscribe():
			if match(result) {
				return result
			}

		case <-time.After(50 * time.Millisecond):
		}
	}

	t.Fatal("timed out waiting for the expected configuration update")
	return Config{}
}

// TestFileStorage_WatchDeliversChanges is the base case: an in-place write to the file is
// noticed and published.
func TestFileStorage_WatchDeliversChanges(t *testing.T) {

	storage := newFileTestStorage(t)

	go storage.watch()

	// Writes repeat until delivery because the watcher opens asynchronously: a write landing
	// before the first watchOnce has called Add produces no event at all.
	deadline := time.Now().Add(10 * time.Second)

	for attempt := 1; time.Now().Before(deadline); attempt++ {

		writeFileConfig(t, storage, "changed@example.com")

		found := false

		for waitUntil := time.Now().Add(500 * time.Millisecond); time.Now().Before(waitUntil) && !found; {
			select {
			case result := <-storage.Subscribe():
				found = result.AdminEmail == "changed@example.com"
			case <-time.After(50 * time.Millisecond):
			}
		}

		if found {
			return
		}
	}

	t.Fatal("the file watcher never delivered the changed configuration")
}

// TestFileStorage_WatchSurvivesAtomicRenameSave is the regression test this file exists for.
// Editors (vim, VS Code, sed -i) save by writing a temp file and renaming it over the original,
// which detaches an fsnotify watch from the file: the OLD single-shot watcher received the
// Rename event, kept watching a dead inode, and never saw another change for the life of the
// process.  The supervised loop must reopen the watch on the new inode and keep delivering.
func TestFileStorage_WatchSurvivesAtomicRenameSave(t *testing.T) {

	storage := newFileTestStorage(t)

	go storage.watch()

	// First rename-save: proves the watch is live, and detaches it
	renameFileConfig(t, storage, "first-rename@example.com")

	awaitFileConfig(t, storage, 10*time.Second, func(value Config) bool {
		return value.AdminEmail == "first-rename@example.com"
	})

	// Second rename-save: everything after this point was lost under the old implementation.
	// Retried because the reopen (with backoff) races this write, and a rename that lands
	// while no watch is attached produces no event -- only a later one does.
	deadline := time.Now().Add(15 * time.Second)

	for attempt := 1; time.Now().Before(deadline); attempt++ {

		renameFileConfig(t, storage, "second-rename@example.com")

		found := false

		for waitUntil := time.Now().Add(time.Second); time.Now().Before(waitUntil) && !found; {
			select {
			case result := <-storage.Subscribe():
				found = result.AdminEmail == "second-rename@example.com"
			case <-time.After(50 * time.Millisecond):
			}
		}

		if found {
			return
		}
	}

	t.Fatal("the watcher never recovered from an atomic-rename save")
}

// TestFileStorage_WatchOnceErrsOnMissingPath pins the failure mode that used to end watching
// forever: Add on a path that does not exist.  watchOnce must return the error (so the
// supervisor can back off and retry), never swallow it.
func TestFileStorage_WatchOnceErrsOnMissingPath(t *testing.T) {

	storage := FileStorage{
		source:        ConfigSourceCommandLine,
		location:      filepath.Join(t.TempDir(), "does-not-exist.json"),
		updateChannel: newUpdateChannel(),
		closeChannel:  make(chan struct{}),
		closeOnce:     &sync.Once{},
	}

	t.Cleanup(storage.Close)

	progressed, err := storage.watchOnce(false)

	require.Error(t, err)
	require.False(t, progressed)
}

// TestFileStorage_CloseStopsTheWatcher pins that the supervised loop is still stoppable.  A loop
// that cannot end except by Close is only correct if Close actually ends it.
func TestFileStorage_CloseStopsTheWatcher(t *testing.T) {

	storage := newFileTestStorage(t)

	finished := make(chan struct{})

	go func() {
		defer close(finished)
		storage.watch()
	}()

	// Give the watcher time to open, so Close interrupts a LIVE watch
	time.Sleep(200 * time.Millisecond)
	storage.Close()

	select {
	case <-finished:
	case <-time.After(10 * time.Second):
		t.Fatal("watch() did not return after Close()")
	}
}

// TestFileStorage_ResynchronizeReadsTheFile pins the reopen behavior: a watcher that was down
// may have missed writes, so watchOnce(resynchronize=true) must publish the file's CURRENT
// contents before waiting for new events.
func TestFileStorage_ResynchronizeReadsTheFile(t *testing.T) {

	storage := newFileTestStorage(t)

	// Change the file while NO watcher exists -- the gap a resynchronize has to cover
	writeFileConfig(t, storage, "written-in-the-gap@example.com")

	go func() {
		_, _ = storage.watchOnce(true)
	}()

	result := awaitFileConfig(t, storage, 10*time.Second, func(value Config) bool {
		return value.AdminEmail == "written-in-the-gap@example.com"
	})

	require.Equal(t, "written-in-the-gap@example.com", result.AdminEmail)
}

/******************************************
 * Constructor Tests
 *
 * Reachable at all only because NewFileStorage returns
 * errors instead of calling os.Exit.
 ******************************************/

// TestNewFileStorage_BootstrapsWithSetup pins the --setup rule: a missing configuration file is
// not an error when setup was requested -- a default configuration is written to disk, published,
// and served.
func TestNewFileStorage_BootstrapsWithSetup(t *testing.T) {

	path := filepath.Join(t.TempDir(), "config.json")

	args := CommandLineArgs{
		Source:   ConfigSourceCommandLine,
		Location: "file://" + path,
		Setup:    true,
	}

	storage, err := NewFileStorage(&args)
	require.NoError(t, err)

	t.Cleanup(storage.Close)

	// The default configuration was written to disk...
	_, statErr := os.Stat(path)
	require.NoError(t, statErr, "--setup must create the configuration file")

	// ...and published to the subscription
	result := awaitFileConfig(t, storage, 10*time.Second, func(value Config) bool {
		return value.HTTPPort == DefaultConfig().HTTPPort
	})

	require.Equal(t, ConfigSourceCommandLine, result.Source)
}

// TestNewFileStorage_MissingFileWithoutSetupErrs pins the boot refusal: without --setup, a
// missing configuration is an error carrying the "re-run with --setup" guidance -- returned to
// main, never an exit from inside this package.
func TestNewFileStorage_MissingFileWithoutSetupErrs(t *testing.T) {

	args := CommandLineArgs{
		Source:   ConfigSourceCommandLine,
		Location: "file://" + filepath.Join(t.TempDir(), "missing.json"),
	}

	_, err := NewFileStorage(&args)

	require.Error(t, err)
	require.Contains(t, err.Error(), "--setup", "the error must tell the operator how to fix it")
}

// TestNewFileStorage_CorruptFileErrs pins the unusable-file branch: a file that exists but
// cannot be parsed is an error, with or without --setup -- overwriting a corrupt (but real)
// configuration with defaults would destroy the operator's settings.
func TestNewFileStorage_CorruptFileErrs(t *testing.T) {

	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte("{this is not valid hjson"), 0600))

	for _, setup := range []bool{false, true} {

		args := CommandLineArgs{
			Source:   ConfigSourceCommandLine,
			Location: "file://" + path,
			Setup:    setup,
		}

		_, err := NewFileStorage(&args)
		require.Error(t, err, "a corrupt file must error (setup=%t)", setup)
	}
}

// TestLoad_RejectsUnknownScheme pins config.Load's contract: an unsupported location scheme is
// an error for main to act on, not a decision this package makes.
func TestLoad_RejectsUnknownScheme(t *testing.T) {

	for _, location := range []string{"", "ftp://example.com/config", "config.json", "http://example.com"} {

		args := CommandLineArgs{Location: location}

		_, err := Load(&args)
		require.Error(t, err, "location: %q", location)
	}
}

// TestFileStorage_WriteRejectsStaleRevision pins the file half of the compare-and-swap: a save
// built from a stale base is rejected, best-effort, instead of silently overwriting the other
// change.  The realistic race on single-server file storage is two setup-console tabs.
func TestFileStorage_WriteRejectsStaleRevision(t *testing.T) {

	storage := newFileTestStorage(t) // the helper's initial write leaves the file at revision 1

	base := DefaultConfig() // Revision 0: stale
	base.AdminEmail = "tab-two@example.com"

	_, err := storage.Write(base)

	require.Error(t, err)
	require.True(t, derp.IsConflict(err), "a stale write must surface as a 409")

	// The file's contents survived untouched
	current, err := storage.load()
	require.NoError(t, err)
	require.Equal(t, "initial@example.com", current.AdminEmail)
}

// TestFileStorage_WriteThreadsRevisions pins the happy path: writes that base each save on the
// previously STORED version succeed, and the revision climbs.
func TestFileStorage_WriteThreadsRevisions(t *testing.T) {

	storage := newFileTestStorage(t)

	current, err := storage.load()
	require.NoError(t, err)

	for range 3 {
		current.AdminEmail = "next@example.com"
		current, err = storage.Write(current)
		require.NoError(t, err)
	}

	require.Equal(t, int64(4), current.Revision, "one bootstrap write plus three saves")
}

// TestFileStorage_WriteAcceptsLegacyFile pins the migration leg: a configuration file written
// before revisions existed has no revision field, loads as Revision 0, and must accept its
// first guarded write.
func TestFileStorage_WriteAcceptsLegacyFile(t *testing.T) {

	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"adminEmail": "legacy@example.com"}`), 0600))

	storage := FileStorage{
		source:        ConfigSourceCommandLine,
		location:      path,
		updateChannel: newUpdateChannel(),
		closeChannel:  make(chan struct{}),
		closeOnce:     &sync.Once{},
	}

	t.Cleanup(storage.Close)

	loaded, err := storage.load()
	require.NoError(t, err)
	require.Equal(t, int64(0), loaded.Revision)

	loaded.AdminEmail = "upgraded@example.com"
	stored, err := storage.Write(loaded)

	require.NoError(t, err)
	require.Equal(t, int64(1), stored.Revision)
}
