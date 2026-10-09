package server

import (
	"sync"
	"testing"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/tools/secretcheck"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests cover the server factory's configuration, which is the point where a background
// reload meets every in-flight HTTP request.  They are written to be run with -race: the
// assertions confirm the semantics, but the race detector is what proves the publication.

// setTestConfig replaces the factory's configuration the way a reload does: under reloadLock,
// which every writer of the wiring is required to hold.
func setTestConfig(factory *factoryCore, value config.Config) {
	factory.reloadLock.Lock()
	defer factory.reloadLock.Unlock()
	factory.setConfigLocked(value)
}

// mutateTestConfig runs a persisted read-modify-write under reloadLock, the way putDomain and
// DeleteDomain do.
func mutateTestConfig(factory *factoryCore, fn func(*config.Config)) error {
	factory.reloadLock.Lock()
	defer factory.reloadLock.Unlock()
	return factory.mutateConfigLocked(fn)
}

// stubStorage is an in-memory config.Storage that honors the same compare-and-swap contract as
// the real engines, so the factory's conflict handling can be tested without a database.
type stubStorage struct {
	lock   sync.Mutex
	stored config.Config
}

// Subscribe returns a channel that never delivers; these tests drive the factory directly.
func (stub *stubStorage) Subscribe() <-chan config.Config {
	return make(chan config.Config)
}

// Read returns the configuration as currently stored.
func (stub *stubStorage) Read() (config.Config, error) {
	stub.lock.Lock()
	defer stub.lock.Unlock()
	return stub.stored.Copy(), nil
}

// Write applies the same CAS rule as the real engines: match the stored revision or 409.
func (stub *stubStorage) Write(value config.Config) (config.Config, error) {

	stub.lock.Lock()
	defer stub.lock.Unlock()

	if value.Revision != stub.stored.Revision {
		return config.Config{}, derp.Conflict("stubStorage.Write", "The configuration was changed by another server")
	}

	stored := value.Copy()
	stored.Revision++
	stub.stored = stored

	return stored.Copy(), nil
}

// Close does nothing; the stub holds no resources.
func (stub *stubStorage) Close() {}

// TestFactoryConfig_ConcurrentReadsAndReloads reads the configuration from many goroutines while
// a reload replaces it, and must run clean under -race
func TestFactoryConfig_ConcurrentReadsAndReloads(t *testing.T) {

	// Config is a dozen words of slices, maps, and strings, wide enough to tear.  Against an
	// unguarded field this fails under -race; against published wiring it is quiet.
	factory := &factoryCore{}
	setTestConfig(factory, config.DefaultConfig())

	var waitGroup sync.WaitGroup

	// Reloaders: what the storage subscription goroutine does
	for writer := range 4 {
		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()

			for index := range 200 {
				value := config.DefaultConfig()
				value.HTTPPort = 8000 + writer
				value.AdminEmail = "admin@example.com"
				value.Domains.Put(config.Domain{DomainID: "1", Hostname: "one.example.com"})
				value.ActivityPubCache["database"] = "cache"
				value.LogSlowQueries = index
				setTestConfig(factory, value)
			}
		}()
	}

	// Readers: what every request handler does
	for range 8 {
		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()

			for range 200 {
				// assert, not require: require fails via runtime.Goexit, which is only legal
				// on the test goroutine
				result := factory.Config()
				assert.NotEmpty(t, result.AttachmentCache)

				factory.ListDomains()
				factory.AllowPrivateIPs()
				factory.IsReadyForDomains()
			}
		}()
	}

	waitGroup.Wait()
}

// TestFactoryConfig_ReturnsADeepCopy requires that a Config handed out shares no maps with the
// live configuration
func TestFactoryConfig_ReturnsADeepCopy(t *testing.T) {

	// The setup console's handlers edit a Config in place before saving, so a shared map would
	// change the running server, unlocked, before the operator saves.
	original := config.DefaultConfig()
	original.AdminEmail = "original@example.com"
	original.Domains.Put(config.Domain{DomainID: "1", Hostname: "original.example.com"})

	factory := &factoryCore{}
	setTestConfig(factory, original)

	// Edit the returned value exactly the way a form handler does
	scratch := factory.Config()
	scratch.AdminEmail = "scratch@example.com"
	scratch.AttachmentCache["location"] = "/tmp/scratch"
	scratch.ActivityPubCache["connectString"] = "mongodb://scratch"
	scratch.Domains[0].Hostname = "scratch.example.com"

	// The live configuration must be untouched
	live := factory.Config()
	require.Equal(t, "original@example.com", live.AdminEmail)
	require.Equal(t, "./.emissary/cache", live.AttachmentCache.GetString("location"))
	require.Empty(t, live.ActivityPubCache.GetString("connectString"))
	require.Equal(t, "original.example.com", live.Domains[0].Hostname)
}

// TestFactoryConfig_CopiesOnTheWayIn requires that a Config handed to the factory is copied, not
// stored
func TestFactoryConfig_CopiesOnTheWayIn(t *testing.T) {

	// The caller usually keeps a reference, since the setup console saves the struct its handler
	// was editing, so storing it would alias the live configuration to that scratch space.
	incoming := config.DefaultConfig()
	incoming.AdminEmail = "incoming@example.com"

	factory := &factoryCore{}
	setTestConfig(factory, incoming)

	// The caller keeps editing its own copy after handing it over
	incoming.AdminEmail = "edited-after-the-fact@example.com"
	incoming.AttachmentCache["location"] = "/tmp/edited"

	live := factory.Config()
	require.Equal(t, "incoming@example.com", live.AdminEmail)
	require.Equal(t, "./.emissary/cache", live.AttachmentCache.GetString("location"))
}

// TestFactoryConfig_ListDomainsIsNotLive pins that the domain list is cloned too.  Returning the
// live slice hands the caller a window that the next configuration reload writes straight through.
func TestFactoryConfig_ListDomainsIsNotLive(t *testing.T) {

	original := config.DefaultConfig()
	original.Domains.Put(config.Domain{DomainID: "1", Hostname: "one.example.com"})

	factory := &factoryCore{}
	setTestConfig(factory, original)

	domains := factory.ListDomains()
	require.Equal(t, 1, len(domains))

	domains[0].Hostname = "hijacked.example.com"

	require.Equal(t, "one.example.com", factory.ListDomains()[0].Hostname)
}

// TestFactoryConfig_MutateMergesIntoTheLatest requires the read-modify-write helper to edit the
// CURRENT configuration, not a snapshot the caller took earlier
func TestFactoryConfig_MutateMergesIntoTheLatest(t *testing.T) {

	// Adding and removing domains use it, so editing a snapshot would silently revert every change
	// that arrived from another node in the meantime.
	factory := &factoryCore{}
	factory.storage = &stubStorage{}
	setTestConfig(factory, config.DefaultConfig())

	// A reload arrives from another node, changing something unrelated
	reloaded := config.DefaultConfig()
	reloaded.AdminEmail = "from-another-node@example.com"
	setTestConfig(factory, reloaded)

	// Now add a domain
	require.NoError(t, mutateTestConfig(factory, func(value *config.Config) {
		value.Domains.Put(config.Domain{DomainID: "1", Hostname: "one.example.com"})
	}))

	// The live configuration carries BOTH changes
	require.Equal(t, "from-another-node@example.com", factory.Config().AdminEmail)
	require.Equal(t, 1, len(factory.Config().Domains))
}

// TestFactoryConfig_MutateRebasesOnConflict requires a mutation whose write conflicts to rebase
// on the stored configuration and re-apply it, so the result carries both changes
func TestFactoryConfig_MutateRebasesOnConflict(t *testing.T) {

	// Before the revision guard, this sequence silently destroyed the other node's change, up to
	// and including a domain's MasterKey, which lives nowhere else.

	// The stored configuration is two revisions ahead of what this factory has seen, and
	// carries another node's change
	other := config.DefaultConfig()
	other.AdminEmail = "other-node@example.com"
	other.Revision = 2

	factory := &factoryCore{}
	factory.storage = &stubStorage{stored: other}

	stale := config.DefaultConfig()
	stale.AdminEmail = "stale@example.com"
	setTestConfig(factory, stale) // Revision 0: two behind the store

	// Add a domain from the stale base
	require.NoError(t, mutateTestConfig(factory, func(value *config.Config) {
		value.Domains.Put(config.Domain{DomainID: "1", Hostname: "one.example.com"})
	}))

	// The stored document has BOTH the other node's change and ours -- nothing was lost
	stored, err := factory.storage.Read()
	require.NoError(t, err)
	require.Equal(t, "other-node@example.com", stored.AdminEmail, "the rebase must keep the other node's change")
	require.Equal(t, 1, len(stored.Domains), "the rebase must re-apply our change")
	require.Equal(t, int64(3), stored.Revision)

	// ...and this factory's live configuration converged on the stored version
	require.Equal(t, "other-node@example.com", factory.Config().AdminEmail)
}

// TestFactoryConfig_UpdateConflictKeepsLocalConfig requires a rejected whole-form save to leave
// this node's live configuration untouched, and never to rebase
func TestFactoryConfig_UpdateConflictKeepsLocalConfig(t *testing.T) {

	// Replaying a human's edit over someone else's change is the silent overwrite the revision
	// prevents.  Keeping local values lets the console re-render them for the human to reconcile.

	// The store is one revision ahead of this factory
	other := config.DefaultConfig()
	other.AdminEmail = "winner@example.com"
	other.Revision = 1

	factory := &factoryCore{}
	factory.storage = &stubStorage{stored: other}

	local := config.DefaultConfig()
	local.AdminEmail = "local@example.com"
	setTestConfig(factory, local) // Revision 0: stale

	// A form save built from the stale base must be REJECTED as a conflict...
	edited := factory.Config()
	edited.AdminEmail = "form-edit@example.com"

	err := factory.UpdateConfig(edited)
	require.Error(t, err)
	require.True(t, derp.IsConflict(err), "a stale form save must surface as a 409")

	// ...leaving both the store and this node's live configuration untouched
	stored, readErr := factory.storage.Read()
	require.NoError(t, readErr)
	require.Equal(t, "winner@example.com", stored.AdminEmail)
	require.Equal(t, "local@example.com", factory.Config().AdminEmail)
}

// faultyStorage is a stubStorage whose Read and Write fail on demand, and which counts its writes
type faultyStorage struct {
	stubStorage
	writeError error // returned by every Write, when set
	readError  error // returned by every Read, when set
	writes     int   // how many times Write has been called
}

// Write fails with writeError when it is set, and otherwise behaves like stubStorage
func (storage *faultyStorage) Write(value config.Config) (config.Config, error) {

	storage.writes++

	if storage.writeError != nil {
		return config.Config{}, storage.writeError
	}

	return storage.stubStorage.Write(value)
}

// Read fails with readError when it is set, and otherwise behaves like stubStorage
func (storage *faultyStorage) Read() (config.Config, error) {

	if storage.readError != nil {
		return config.Config{}, storage.readError
	}

	return storage.stubStorage.Read()
}

// TestFactoryConfig_UpdateFailure verifies that a failed save is wrapped, is not mistaken for a
// conflict, and leaves this node's configuration alone.
func TestFactoryConfig_UpdateFailure(t *testing.T) {

	factory := &factoryCore{}
	factory.storage = &faultyStorage{writeError: derp.Internal("test", "Storage is down")}
	setTestConfig(factory, config.DefaultConfig())

	edited := configWithDomains(config.Domain{
		DomainID:      "1",
		Hostname:      "one.example.com",
		ConnectString: "mongodb://user:db-password-secret@127.0.0.1:59999/",
		MasterKey:     testMasterKey,
	})

	err := factory.UpdateConfig(edited)

	require.Error(t, err)
	require.False(t, derp.IsConflict(err))
	require.Equal(t, "server.factory.UpdateConfig", derp.Location(err))
	require.Empty(t, factory.ListDomains(), "a failed save must not be published")

	// The configuration it failed to save carries every domain's secrets
	secretcheck.RequireAbsent(t, err, testMasterKey)
	secretcheck.RequireAbsent(t, err, "db-password-secret")
}

// TestFactoryConfig_MutateGivesUpAfterThreeConflicts pins the retry bound
func TestFactoryConfig_MutateGivesUpAfterThreeConflicts(t *testing.T) {

	storage := &faultyStorage{writeError: derp.Conflict("test", "Always conflicting")}

	factory := &factoryCore{}
	factory.storage = storage
	setTestConfig(factory, config.DefaultConfig())

	err := mutateTestConfig(factory, func(value *config.Config) {
		value.Domains.Put(config.Domain{DomainID: "1", Hostname: "one.example.com"})
	})

	require.Error(t, err)
	require.True(t, derp.IsConflict(err))
	require.Equal(t, 3, storage.writes)
	require.Empty(t, factory.ListDomains())
}

// TestFactoryConfig_MutateDoesNotRetryOtherErrors verifies that only a conflict is retried
func TestFactoryConfig_MutateDoesNotRetryOtherErrors(t *testing.T) {

	storage := &faultyStorage{writeError: derp.Internal("test", "Storage is down")}

	factory := &factoryCore{}
	factory.storage = storage
	setTestConfig(factory, config.DefaultConfig())

	err := mutateTestConfig(factory, func(value *config.Config) {
		value.AdminEmail = "changed@example.com"
	})

	require.Error(t, err)
	require.False(t, derp.IsConflict(err))
	require.Equal(t, 1, storage.writes)
	require.Empty(t, factory.Config().AdminEmail)
}

// TestFactoryConfig_MutateStopsWhenTheRebaseFails verifies that a conflict followed by a failed
// re-read returns the read error instead of retrying blind.
func TestFactoryConfig_MutateStopsWhenTheRebaseFails(t *testing.T) {

	storage := &faultyStorage{
		writeError: derp.Conflict("test", "Conflicting"),
		readError:  derp.Internal("test", "Storage is down"),
	}

	factory := &factoryCore{}
	factory.storage = storage
	setTestConfig(factory, config.DefaultConfig())

	err := mutateTestConfig(factory, func(value *config.Config) {
		value.AdminEmail = "changed@example.com"
	})

	require.Error(t, err)
	require.False(t, derp.IsConflict(err), "the read failure, not the conflict, is what stopped the save")
	require.Equal(t, 1, storage.writes)
}

// TestFactoryConfig_UpdateSucceeds verifies that a save publishes the STORED version, whose
// revision the next save must match.
func TestFactoryConfig_UpdateSucceeds(t *testing.T) {

	factory := &factoryCore{}
	factory.storage = &stubStorage{}
	setTestConfig(factory, config.DefaultConfig())

	edited := factory.Config()
	edited.AdminEmail = "edited@example.com"

	require.NoError(t, factory.UpdateConfig(edited))
	require.Equal(t, "edited@example.com", factory.Config().AdminEmail)
	require.Equal(t, int64(1), factory.Config().Revision)

	// The next save builds on the published revision, so it does not conflict
	again := factory.Config()
	again.AdminEmail = "again@example.com"

	require.NoError(t, factory.UpdateConfig(again))
	require.Equal(t, int64(2), factory.Config().Revision)
}
