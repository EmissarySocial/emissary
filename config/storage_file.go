package config

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/benpate/derp"
	"github.com/fsnotify/fsnotify"
	"github.com/hjson/hjson-go/v4"
	"github.com/rs/zerolog/log"
)

// FileStorage is a file-based storage engine for the server configuration
type FileStorage struct {
	source        string
	location      string
	updateChannel updateChannel
	closeChannel  chan struct{}
	closeOnce     *sync.Once
}

// NewFileStorage creates a fully initialized FileStorage instance.
func NewFileStorage(args *CommandLineArgs) (FileStorage, error) {

	// It never ends the process: every failure comes back as an error carrying the operator-facing
	// guidance, and main decides what to do with it (see config.Load).  A missing configuration is
	// only an error without --setup; with it, a default configuration is written and served.

	const location = "config.NewFileStorage"

	fileLocation := strings.TrimPrefix(args.Location, "file://")

	// Create a new FileStorage instance
	storage := FileStorage{
		source:        args.Source,
		location:      fileLocation,
		updateChannel: newUpdateChannel(),
		closeChannel:  make(chan struct{}),
		closeOnce:     &sync.Once{},
	}

	// Special rules for the first time we load the configuration file
	config, err := storage.load()

	switch {

	// If the config was read successfully, then NOOP here skips down to the next section.
	case err == nil:

	// If the config was not found, then run in setup mode and create a new default configuration
	case derp.IsNotFound(err):

		// RULE: A missing configuration is an error UNLESS --setup was requested; creating a
		// fresh configuration is exactly what setup mode is for.
		if !args.Setup {
			return FileStorage{}, derp.Wrap(err, location, "The configuration file could not be found. Re-run Emissary with the --setup flag to create one.", fileLocation)
		}

		log.Debug().Msg("Configuration file not found.  Running in setup mode.")

		// Overwrite the configuration with a default value
		config = DefaultConfig()
		config.Source = storage.source
		config.Location = storage.location

		// Save the config to disk, keeping the STORED version (with its stamped revision) so
		// the first console save does not conflict against the bootstrap write
		written, inner := storage.Write(config)

		if inner != nil {
			return FileStorage{}, derp.Wrap(inner, location, "Unable to write a new configuration file", fileLocation)
		}

		config = written

	// Anything but a "Not Found" error means the file exists but cannot be used.
	default:
		return FileStorage{}, derp.Wrap(err, location, "The configuration file could not be read. Check the file for syntax errors.", fileLocation)
	}

	// If we have a valid config, post it to the update channel
	storage.updateChannel.notify(config)

	log.Info().Msg("Loading configuration from file")

	// After the first load, watch for changes to the configuration file and post them to the update channel
	go storage.watch()

	return storage, nil
}

// Subscribe returns a channel that will receive the configuration every time it is updated
func (storage FileStorage) Subscribe() <-chan Config {
	return storage.updateChannel.subscribe()
}

// Close shuts down the filesystem watcher.  Idempotent, like MongoStorage.Close: a second call
// is a no-op, not a panic on a closed channel.
func (storage FileStorage) Close() {
	storage.closeOnce.Do(func() {
		close(storage.closeChannel)
	})
}

/******************************************
 * Filesystem Watcher
 ******************************************/

// watch supervises a filesystem watcher on the configuration file for the whole life of the
// process, reopening it whenever it dies.  It is the same supervision that MongoStorage.watch
// applies to its change stream, for the same reason:
func (storage FileStorage) watch() {

	// RULE: This loop MUST NOT be able to end except by Close().  A single-shot watcher has several
	// quiet deaths -- fsnotify construction can fail, Add can fail, and an atomic-rename save (vim
	// and many other editors write a temp file and rename it over the original) detaches the watch
	// from the file's new inode, after which no event ever arrives again.  Any of those used to
	// leave the process running on a frozen configuration, with nothing in the log, until reboot.

	const location = "config.FileStorage.watch"

	// `failures` counts CONSECUTIVE failed attempts, so it lives across iterations: it drives
	// the backoff, and resets whenever a watcher proves healthy by delivering an event.
	for failures := 0; ; {

		// RULE: Close is the ONLY way out of this loop.
		if storage.isClosed() {
			return
		}

		progressed, err := storage.watchOnce(failures > 0)

		if storage.isClosed() {
			return
		}

		// A watcher that delivered events was healthy; reset the backoff so a long-lived watch
		// that finally drops reconnects promptly instead of inheriting an old penalty.
		if progressed {
			failures = 0
		}

		switch {

		case err != nil:
			derp.Report(derp.Wrap(err, location, "Configuration file watcher failed. Reopening."))

		// The quiet case: the watch detached (rename save) or its channel closed, with no error.
		default:
			log.Debug().Str("loc", location).Msg("Configuration file watcher ended. Reopening.")
		}

		failures++

		// RULE: Never hot-loop against a broken filesystem.  Sleep, but stay closable.
		select {
		case <-storage.closeChannel:
			return
		case <-time.After(watchRetryDelay(failures)):
		}
	}
}

// watchOnce opens one filesystem watcher and pumps its events until it dies.  It reports whether
// any event was delivered, and the terminating error (nil when the watch simply detached).
func (storage FileStorage) watchOnce(resynchronize bool) (bool, error) {

	const location = "config.FileStorage.watchOnce"

	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		return false, derp.Wrap(err, location, "Creating filesystem watcher")
	}

	defer derp.ReportFunc(watcher.Close)

	if err := watcher.Add(storage.location); err != nil {
		return false, derp.Wrap(err, location, "Watching for changes to configuration", storage.location)
	}

	// A reopened watcher may have missed changes while it was down, so re-read the file now.
	// Reloads are idempotent, which makes this cheap insurance against a silent gap.
	if resynchronize {
		storage.reload()
	}

	for progressed := false; ; {
		select {

		case <-storage.closeChannel:
			return progressed, nil

		case event, ok := <-watcher.Events:

			// A closed channel means the watcher itself died; hand control back for a reopen
			if !ok {
				return progressed, nil
			}

			progressed = true
			storage.reload()

			// RULE: A Remove or Rename means the path we are watching no longer names the file
			// -- the classic atomic-rename save.  The watch is now attached to a dead inode, so
			// return and let the supervisor re-Add the path, or no later save is ever seen.
			if event.Op&(fsnotify.Remove|fsnotify.Rename) != 0 {
				return progressed, nil
			}

		case err, ok := <-watcher.Errors:

			if !ok {
				return progressed, nil
			}

			return progressed, derp.Wrap(err, location, "Watching for changes to configuration", storage.location)
		}
	}
}

// reload reads the configuration file and publishes it, ignoring an empty or half-written file.
// Editors fire several events per save, so any one of them may catch the file mid-write; the
// next event (or the resynchronize pass after a reopen) delivers the settled contents.
func (storage FileStorage) reload() {

	const location = "config.FileStorage.reload"

	config, err := storage.load()

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Loading the updated config from", storage.location))
		return
	}

	if config.IsEmpty() {
		return
	}

	storage.updateChannel.notify(config)
}

// isClosed reports whether Close has been called on this storage
func (storage FileStorage) isClosed() bool {

	select {
	case <-storage.closeChannel:
		return true
	default:
		return false
	}
}

// load reads the configuration from the filesystem and
// creates a default configuration if the file is missing
func (storage FileStorage) load() (Config, error) {

	result := NewConfig()

	// Try to load the configuration file from disk
	data, err := os.ReadFile(storage.location)

	if err != nil {
		return Config{}, derp.Wrap(err, "config.FileStorage.load", "Reading configuration", derp.WithNotFound())
	}

	if err := hjson.Unmarshal(data, &result); err != nil {
		return Config{}, derp.Internal("config.FileStorage.load", "Error unmarshaling configuration", derp.WithWrappedValue(err))
	}

	result.Source = storage.source
	result.Location = storage.location

	// RULE: Warn loudly about domains that cannot encrypt vault data (BUG-110)
	result.ReportInvalidMasterKeys()

	return result, nil
}

// Read returns the configuration as currently stored.  It is part of the Storage interface;
// callers use it to rebase a read-modify-write after Write reports a revision conflict.
func (storage FileStorage) Read() (Config, error) {
	return storage.load()
}

// Write persists the configuration and returns it as stored, with its Revision incremented.
func (storage FileStorage) Write(config Config) (Config, error) {

	const location = "config.FileStorage.Write"

	// RULE: Same compare-and-swap contract as MongoStorage.Write, held BEST-EFFORT: the check and
	// the write are not atomic without file locking, and file storage is single-server by design,
	// so the realistic race this closes is two setup-console tabs saving from different bases.

	// RULE: Verify the revision against the file before replacing it.  A missing file is the
	// first write; an unreadable file is an error rather than a clobber -- a corrupt-but-real
	// configuration holds settings (like domain MasterKeys) that a blind overwrite would
	// destroy, and recovery is the operator's call.
	switch current, err := storage.load(); {

	case err == nil:
		if current.Revision != config.Revision {
			return Config{}, derp.Conflict(location, "The configuration was changed by another process. Your change was NOT saved. Reload and try again.")
		}

	case derp.IsNotFound(err):
		// First write: nothing to conflict with

	default:
		return Config{}, derp.Wrap(err, location, "Unable to verify the current configuration file", storage.location)
	}

	// The stored file always carries the NEXT revision
	stored := config
	stored.Revision = config.Revision + 1

	// Marshal the configuration to JSON
	data, err := json.MarshalIndent(stored, "", "    ")

	if err != nil {
		return Config{}, derp.Wrap(err, location, "Marshaling configuration")
	}

	// Try to write the configuration to disk.
	// RULE: Mode 0600 (owner read/write only) because this file contains secrets
	// (SMTP password, database credentials, certificate locations).
	if err := os.WriteFile(storage.location, data, 0600); err != nil {
		return Config{}, derp.Wrap(err, location, "Writing configuration")
	}

	return stored, nil
}
