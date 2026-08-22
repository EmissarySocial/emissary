package server

import (
	"net/http"
	"sync"
	"testing"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/rosetta/mapof"
	"github.com/puzpuzpuz/xsync/v4"
	"github.com/realclientip/realclientip-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests cover the point where a background configuration reload meets every in-flight HTTP
// request.  They are written to be run with -race: the assertions describe the semantics, but the
// race detector is what proves the publication is safe.
//
// RULE for anyone adding a field to factoryCore: if a reload writes it and a request reads it, it
// belongs in `wiring`, not on the struct.  These tests fail loudly when it does not.

// newConcurrencyTestFactory returns a factoryCore complete enough to run the rewiring
// half of a reload, with no reachable database behind it.
func newConcurrencyTestFactory(t *testing.T) *factoryCore {

	t.Helper()

	result := newTestFactoryCore()
	result.domains = xsync.NewMap[string, *service.Factory]()

	// Both real constructors publish a strategy before the factory serves anything, so a reader
	// never legitimately sees a nil one
	result.rewire(func(value *wiring) {
		value.clientIPStrategy = realclientip.RemoteAddrStrategy{}
	})

	t.Cleanup(func() {
		result.currentWiring().queue.Stop()
	})

	return result
}

// TestWiring_ConcurrentReloadAndReads is the regression test for the original defect: the
// common database, the queue, and the client-IP strategy were plain fields, written by the
// configuration-reload goroutine while request goroutines read them.
//
// Run with -race.  The reader half is unchanged from what a request handler always did; it is
// the writer half that used to assign these fields in place, and the detector fired on the pair.
func TestWiring_ConcurrentReloadAndReads(t *testing.T) {

	factory := newConcurrencyTestFactory(t)

	var waitGroup sync.WaitGroup

	// The reloader: what the storage subscription goroutine does to this wiring
	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		for index := range 200 {

			// Alternate the settings so that every pass actually swaps the connection
			database := "reload-a"

			if index%2 == 1 {
				database = "reload-b"
			}

			factory.reloadLock.Lock()
			_, err := factory.refreshCommonDatabase(lifecycleConnection("59999", database), false)
			assert.NoError(t, err)
			factory.refreshQueue(false)
			factory.setConfigLocked(config.DefaultConfig())
			factory.rewireLocked(func(value *wiring) {
				value.clientIPStrategy = realclientip.RemoteAddrStrategy{}
			})
			factory.reloadLock.Unlock()
		}
	}()

	// The readers: what request handlers do
	request, err := http.NewRequest(http.MethodGet, "http://example.com/", nil)
	require.NoError(t, err)
	request.RemoteAddr = "192.0.2.1:1234"

	for range 8 {
		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()

			for range 200 {
				// assert, not require: require fails via runtime.Goexit, which is only legal on
				// the test goroutine
				assert.NotNil(t, factory.Queue())
				assert.NotEmpty(t, factory.ClientIP(request))

				factory.CommonDatabase()
				factory.Config()
				factory.ListDomains()
			}
		}()
	}

	waitGroup.Wait()
}

// TestWiring_ReadersSeeWholeGenerations is the property that a lock over each field
// separately would NOT give: a reader loads one generation and every field in it came from the
// same publish.  Here the connection's URI and database name are always written together, so a
// reader that could see one from before a swap and the other from after would be reading a
// connection that never existed.
func TestWiring_ReadersSeeWholeGenerations(t *testing.T) {

	factory := newConcurrencyTestFactory(t)

	// Two internally-consistent pairs.  Any mix of the two is a torn read.
	pairs := []mapof.String{
		{"connectString": "mongodb://alpha", "database": "alpha"},
		{"connectString": "mongodb://beta", "database": "beta"},
	}

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		for index := range 500 {
			pair := pairs[index%2]
			factory.rewire(func(value *wiring) {
				value.commonDatabaseURI = pair.GetString("connectString")
				value.commonDatabaseName = pair.GetString("database")
			})
		}
	}()

	for range 8 {
		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()

			for range 500 {

				current := factory.currentWiring()

				if current.commonDatabaseURI == "" {
					continue
				}

				assert.Equal(t, "mongodb://"+current.commonDatabaseName, current.commonDatabaseURI,
					"a reader must never see one generation's URI beside another's database name")
			}
		}()
	}

	waitGroup.Wait()
}

// TestWiring_PublishedGenerationIsImmutable pins the rule the whole design rests on.  A
// reader may hold its pointer for as long as it likes, so a later publish must build a new value
// rather than write through the old one.
func TestWiring_PublishedGenerationIsImmutable(t *testing.T) {

	factory := newConcurrencyTestFactory(t)

	factory.rewire(func(value *wiring) {
		value.commonDatabaseName = "first"
	})

	// A request goroutine loads the generation and keeps using it...
	held := factory.currentWiring()

	// ...while a reload publishes a new one
	factory.rewire(func(value *wiring) {
		value.commonDatabaseName = "second"
	})

	require.Equal(t, "first", held.commonDatabaseName, "a published generation must never change under its readers")
	require.Equal(t, "second", factory.currentWiring().commonDatabaseName)
}

// TestWiring_ZeroValueIsUsable pins that currentWiring never returns nil.  Tests build a
// bare factoryCore, and a reader can race ahead of the first publish at boot; neither may panic.
func TestWiring_ZeroValueIsUsable(t *testing.T) {

	factory := factoryCore{}

	result := factory.currentWiring()

	require.NotNil(t, result)
	require.Nil(t, result.commonDatabase)
	require.Nil(t, result.queue)
	require.False(t, result.commonDatabaseVerified)
}

// TestSetupFactory_ConcurrentConfigureAndConnectionChecks reproduces the exact pair from the
// original bug report: the setup console's reload goroutine reconnecting the common database
// while request goroutines read the connection and the configuration.  It is self-triggering in
// production -- a console save writes the config, the change stream echoes it back, and the
// reload lands while the POST that caused it is still running.
func TestSetupFactory_ConcurrentConfigureAndConnectionChecks(t *testing.T) {

	factory := SetupFactory{}
	factory.domains = xsync.NewMap[string, *service.Factory]()

	factory.rewire(func(value *wiring) {
		value.clientIPStrategy = realclientip.RemoteAddrStrategy{}
	})

	// The ping fails against this closed port, which exercises the verify-and-roll-back path --
	// the one that rewrites the whole connection group at once, and so the one most likely to be
	// seen mid-write.
	connection := lifecycleConnection("59999", "setup-race")

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		for range 5 {
			factory.reloadLock.Lock()
			// The error is expected: nothing is listening on that port
			_, _ = factory.refreshCommonDatabase(connection, true)
			factory.reloadLock.Unlock()
		}
	}()

	for range 4 {
		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()

			for range 500 {
				factory.CommonDatabase()
				factory.Config()
				factory.IsReadyForDomains()
			}
		}()
	}

	waitGroup.Wait()
}
