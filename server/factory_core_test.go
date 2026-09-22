package server

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/service"
	derpconsole "github.com/EmissarySocial/emissary/tools/derp-console"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/labstack/echo/v4"
	"github.com/puzpuzpuz/xsync/v4"
	"github.com/realclientip/realclientip-go"
	"github.com/rs/zerolog"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

/******************************************
 * Test Helpers
 ******************************************/

// liveTestConnection is the local MongoDB replica set that this package's integration tests use
const liveTestConnection = "mongodb://localhost:27017/?directConnection=true"

// unreachableConnection names a closed port, so a domain built on it fails within 200ms
const unreachableConnection = "mongodb://127.0.0.1:59999/?directConnection=true&serverSelectionTimeoutMS=200"

// testMasterKey stands in for a domain's MasterKey, so tests can prove it never reaches an error
const testMasterKey = "5ec7e75ec7e75ec7e75ec7e75ec7e75ec7e75ec7e75ec7e75ec7e75ec7e75ec7"

// requireLiveMongo returns a client for the local replica set, skipping the test in -short mode,
// when no server answers, or when the server cannot run change streams and transactions.
func requireLiveMongo(t *testing.T) *mongo.Client {

	t.Helper()

	if testing.Short() {
		t.Skip("Skipping MongoDB integration test in -short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(liveTestConnection))

	if err != nil {
		t.Skip("Skipping: no MongoDB at " + liveTestConnection)
	}

	t.Cleanup(func() {
		_ = client.Disconnect(context.Background()) // nothing left to release if this fails
	})

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		t.Skip("Skipping: no MongoDB at " + liveTestConnection)
	}

	// Change streams and transactions both need a replica set
	if err := client.Database("admin").RunCommand(ctx, bson.D{{Key: "replSetGetStatus", Value: 1}}).Err(); err != nil {
		t.Skip("Skipping: MongoDB at " + liveTestConnection + " is not a replica set")
	}

	return client
}

// newLiveDomainCore returns a factoryCore connected to a throwaway common database on the local
// replica set, plus the configuration of a domain whose own throwaway database lives there too.
func newLiveDomainCore(t *testing.T) (*factoryCore, config.Domain) {

	t.Helper()

	factory := newTestFactoryCore()
	return factory, prepareLiveDomainCore(t, factory)
}

// newLiveSetupFactory is newLiveDomainCore for the setup console's factory
func newLiveSetupFactory(t *testing.T) (*SetupFactory, config.Domain) {

	t.Helper()

	factory := &SetupFactory{}
	factory.rewire(func(value *wiring) {
		value.queue = queue.New()
	})

	return factory, prepareLiveDomainCore(t, &factory.factoryCore)
}

// prepareLiveDomainCore connects the factory to a throwaway common database on the local replica
// set, and returns the configuration of a domain whose own throwaway database lives there too.
func prepareLiveDomainCore(t *testing.T, factory *factoryCore) config.Domain {

	t.Helper()

	client := requireLiveMongo(t)
	suffix := primitive.NewObjectID().Hex()

	// Let watchers stopped by earlier tests finish exiting, so every count starts from zero
	requireWatchersStopped(t, 0, "watchers from an earlier test are still running")

	// The common database gets its own client, because the verify tests disconnect it
	commonClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI(liveTestConnection))
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = commonClient.Disconnect(context.Background()) // a test may have disconnected it already
	})

	factory.domains = xsync.NewMap[string, *service.Factory]()
	factory.storage = &stubStorage{}
	stopQueueOnCleanup(t, factory)
	setTestFilesystems(factory)
	setTestCommonDatabase(factory, commonClient.Database("emissary_servertest_common_"+suffix))

	domainConfig := config.Domain{
		DomainID:      suffix,
		Hostname:      "live-" + suffix + ".example.com",
		ConnectString: liveTestConnection,
		DatabaseName:  "emissary_servertest_" + suffix,
		MasterKey:     config.NewMasterKey(),
	}

	// Drop every database the test may create through the cleanup client, which no test disconnects
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_ = client.Database(domainConfig.DatabaseName).Drop(ctx)
		_ = client.Database(domainConfig.DatabaseName + "_moved").Drop(ctx)
		_ = client.Database("emissary_servertest_common_" + suffix).Drop(ctx)
		_ = client.Database("emissary_servertest_common_" + suffix + "_moved").Drop(ctx)
	})

	return domainConfig
}

// releaseDomainFactory closes a domain factory when the test ends, whether or not the factory is
// still in the registry by then.  Close is safe to call twice.
func releaseDomainFactory(t *testing.T, domain *service.Factory) {
	t.Helper()
	t.Cleanup(domain.Close)
}

// requireEventuallyDisconnected waits for the client to be disconnected, which Close does in the background
func requireEventuallyDisconnected(t *testing.T, client *mongo.Client, message string) {
	t.Helper()
	require.Eventually(t, func() bool {
		return errors.Is(client.Ping(context.Background(), readpref.Primary()), mongo.ErrClientDisconnected)
	}, 10*time.Second, 20*time.Millisecond, message)
}

// setTestFilesystems publishes in-memory filesystems, which every new domain factory requires
func setTestFilesystems(factory *factoryCore) {
	factory.rewire(func(value *wiring) {
		value.attachmentOriginals = afero.NewMemMapFs()
		value.attachmentCache = afero.NewMemMapFs()
		value.exportCache = afero.NewMemMapFs()
	})
}

// reloadDomains runs refreshDomains under reloadLock, as both mode lifecycles do
func reloadDomains(factory *factoryCore, value config.Config) {
	factory.reloadLock.Lock()
	defer factory.reloadLock.Unlock()
	factory.refreshDomains(value)
}

// configWithDomains returns the default configuration carrying the given domains
func configWithDomains(domains ...config.Domain) config.Config {

	result := config.DefaultConfig()

	for _, domain := range domains {
		result.Domains.Put(domain)
	}

	return result
}

// countGoroutinesStartedBy returns how many running goroutines the named function started
func countGoroutinesStartedBy(function string) int {

	// Grow the buffer until the dump of every goroutine fits
	for buffer := make([]byte, 1<<20); ; buffer = make([]byte, 2*len(buffer)) {
		if size := runtime.Stack(buffer, true); size < len(buffer) {
			return strings.Count(string(buffer[:size]), "created by github.com/EmissarySocial/emissary/"+function+" ")
		}
	}
}

// countDomainWatchers returns how many change stream watchers the domain factories are running,
// which are the goroutines that service.Factory.Refresh starts.
func countDomainWatchers() int {
	return countGoroutinesStartedBy("service.(*Factory).Refresh")
}

// startLiveDomain loads one domain through refreshDomains, and returns its factory once the
// background upgrade that Domain.Start launches has finished.
func startLiveDomain(t *testing.T, factory *factoryCore, domainConfig config.Domain) *service.Factory {

	t.Helper()

	reloadDomains(factory, configWithDomains(domainConfig))

	domain, err := factory.ByHostname(domainConfig.Hostname)
	require.NoError(t, err)
	releaseDomainFactory(t, domain)

	// Waited out because that upgrade reads a field Domain.Refresh writes unsynchronized, a race
	// in service that -race would pin on whichever test refreshes the domain next.
	waitForDomainStartup(t)
	return domain
}

// waitForDomainStartup waits for every background upgrade that Domain.Start launched to finish
func waitForDomainStartup(t *testing.T) {
	t.Helper()
	require.Eventually(t, func() bool { return countGoroutinesStartedBy("service.(*Domain).Start") == 0 }, 60*time.Second, 20*time.Millisecond, "the domain's background upgrade never finished")
}

// requireWatchersStopped waits for the watcher count to fall back to `baseline`
func requireWatchersStopped(t *testing.T, baseline int, message string) {
	t.Helper()
	require.Eventually(t, func() bool { return countDomainWatchers() == baseline }, 10*time.Second, 20*time.Millisecond, message)
}

// requireNoSecret fails if the error, encoded the way an error reporter stores it, contains `secret`
func requireNoSecret(t *testing.T, err error, secret string) {

	t.Helper()

	encoded, marshalErr := json.Marshal(err)
	require.NoError(t, marshalErr)
	require.NotContains(t, string(encoded), secret, "error details must not carry secrets")
	require.NotContains(t, err.Error(), secret, "error messages must not carry secrets")
}

// reportRecorder is a derp.Reporter that keeps every error reported to it
type reportRecorder struct {
	lock   sync.Mutex
	errors []error
}

// Report records one error
func (recorder *reportRecorder) Report(err error) {
	recorder.lock.Lock()
	defer recorder.lock.Unlock()
	recorder.errors = append(recorder.errors, err)
}

// reported returns a copy of every error recorded so far
func (recorder *reportRecorder) reported() []error {
	recorder.lock.Lock()
	defer recorder.lock.Unlock()
	return append([]error(nil), recorder.errors...)
}

// recordReports routes every derp.Report to a recorder until the test ends
func recordReports(t *testing.T) *reportRecorder {

	t.Helper()

	recorder := &reportRecorder{}
	derp.SetPlugins(recorder)

	t.Cleanup(func() {
		derp.SetPlugins(derpconsole.New())
	})

	return recorder
}

/******************************************
 * init and Global Services
 ******************************************/

// TestFactoryCore_Init verifies that init builds every global service, in place
func TestFactoryCore_Init(t *testing.T) {

	storage := &stubStorage{}
	factory := &factoryCore{}
	factory.init(storage, embed.FS{})

	t.Cleanup(func() {
		factory.currentWiring().queue.Stop()
	})

	// The registry and the storage are ready
	require.Same(t, storage, factory.storage)
	require.NotNil(t, factory.domains)
	require.Zero(t, factory.domains.Size())

	// An inert placeholder queue is installed, and is not mistaken for the real one
	require.NotNil(t, factory.Queue())
	require.False(t, factory.currentWiring().queueReady)

	// Every accessor reads the service init built on this factory
	require.Same(t, &factory.contentService, factory.Content())
	require.Same(t, &factory.registrationService, factory.Registration())
	require.Same(t, &factory.templateService, factory.Template())
	require.Same(t, &factory.themeService, factory.Theme())
	require.Same(t, &factory.widgetService, factory.Widget())
	require.Same(t, &factory.emailService, factory.Email())
	require.Same(t, &factory.httpCache, factory.HTTPCache())
	require.IsType(t, service.Icons{}, factory.Icons())
	require.IsType(t, service.Filesystem{}, factory.Filesystem())
	require.NotNil(t, factory.DigitalDome())
	require.NotNil(t, factory.EditorJS())
	require.NotNil(t, factory.workingDirectory)

	// The template functions render icons through the icon service
	require.Contains(t, factory.FuncMap(), "icon")
}

/******************************************
 * Domain Registry
 ******************************************/

// TestRangeDomains verifies that the iterator visits every domain factory, and stops when asked
func TestRangeDomains(t *testing.T) {

	factory := testPersonalizedFactory("one.example.com", "two.example.com", "three.example.com")

	visited := 0

	for domain := range factory.RangeDomains() {
		require.NotNil(t, domain)
		visited++
	}

	require.Equal(t, 3, visited)

	// Breaking out of the loop stops the iteration
	visited = 0

	for range factory.RangeDomains() {
		visited++
		break
	}

	require.Equal(t, 1, visited)
}

// TestFindDomain covers the "new" keyword, a configured domain, and an unknown one
func TestFindDomain(t *testing.T) {

	factory := &factoryCore{}
	setTestConfig(factory, configWithDomains(config.Domain{DomainID: "1", Hostname: "one.example.com"}))

	t.Run("New", func(t *testing.T) {
		for _, keyword := range []string{"new", "NEW", "New"} {
			domain, err := factory.FindDomain(keyword)
			require.NoError(t, err, keyword)
			require.NotEmpty(t, domain.DomainID, keyword)
			require.NotEqual(t, "1", domain.DomainID, keyword)
			require.Empty(t, domain.Hostname, keyword)
		}
	})

	t.Run("Found", func(t *testing.T) {
		domain, err := factory.FindDomain("1")
		require.NoError(t, err)
		require.Equal(t, "one.example.com", domain.Hostname)
	})

	t.Run("NotFound", func(t *testing.T) {
		domain, err := factory.FindDomain("missing")
		require.Error(t, err)
		require.True(t, derp.IsNotFound(err))
		require.Empty(t, domain.Hostname)
	})
}

// TestByDomainID covers a served domain, a configured domain with no factory, and an unknown ID
func TestByDomainID(t *testing.T) {

	factory := testPersonalizedFactory("one.example.com")
	setTestConfig(factory, configWithDomains(
		config.Domain{DomainID: "1", Hostname: "one.example.com"},
		config.Domain{DomainID: "2", Hostname: "two.example.com"},
	))

	t.Run("Served", func(t *testing.T) {
		domainConfig, domain, err := factory.ByDomainID("1")
		require.NoError(t, err)
		require.NotNil(t, domain)
		require.Equal(t, "one.example.com", domainConfig.Hostname)
	})

	t.Run("NotServed", func(t *testing.T) {
		domainConfig, domain, err := factory.ByDomainID("2")
		require.Error(t, err)
		require.Equal(t, http.StatusMisdirectedRequest, derp.ErrorCode(err))
		require.Nil(t, domain)
		require.Empty(t, domainConfig.Hostname)
	})

	t.Run("Unknown", func(t *testing.T) {
		domainConfig, domain, err := factory.ByDomainID("missing")
		require.Error(t, err)
		require.True(t, derp.IsNotFound(err))
		require.Nil(t, domain)
		require.Empty(t, domainConfig.Hostname)
	})
}

// TestByRequest resolves domain factories from the Host header, and from echo contexts
func TestByRequest(t *testing.T) {

	factory := testPersonalizedFactory("one.example.com")
	setTestConfig(factory, config.DefaultConfig())

	t.Run("Served", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "http://one.example.com/", nil)

		domain, err := factory.ByRequest(request)
		require.NoError(t, err)
		require.NotNil(t, domain)

		// ByContext reads the same request
		echoContext := echo.New().NewContext(request, httptest.NewRecorder())
		fromContext, err := factory.ByContext(echoContext)
		require.NoError(t, err)
		require.Same(t, domain, fromContext)
	})

	t.Run("NotServed", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "http://two.example.com/", nil)

		domain, err := factory.ByRequest(request)
		require.Error(t, err)
		require.Equal(t, http.StatusMisdirectedRequest, derp.ErrorCode(err))
		require.Nil(t, domain)
	})
}

// TestHostname verifies that X-Forwarded-Host is honored only when the configuration trusts it
func TestHostname(t *testing.T) {

	request := httptest.NewRequest(http.MethodGet, "http://direct.example.com/", nil)
	request.Header.Set("X-Forwarded-Host", "forwarded.example.com")

	unforwarded := httptest.NewRequest(http.MethodGet, "http://direct.example.com/", nil)

	factory := &factoryCore{}

	// Untrusted: the Host header wins, even when the forwarded header is present
	setTestConfig(factory, config.DefaultConfig())
	require.Equal(t, "direct.example.com", factory.Hostname(request))

	// Trusted: the forwarded header wins, but only when it is present
	trusted := config.DefaultConfig()
	trusted.TrustForwardedHost = true
	setTestConfig(factory, trusted)

	require.Equal(t, "forwarded.example.com", factory.Hostname(request))
	require.Equal(t, "direct.example.com", factory.Hostname(unforwarded))
}

// TestByHostname_Normalizes verifies that lookups ignore case, ports, and a leading "www."
func TestByHostname_Normalizes(t *testing.T) {

	factory := testPersonalizedFactory("example.com")

	for _, hostname := range []string{
		"example.com",
		"EXAMPLE.COM",
		"example.com:8080",
		"www.example.com",
		"WWW.example.com", // the "www." must be recognized in any case
		"WWW.EXAMPLE.COM:443",
	} {
		domain, err := factory.ByHostname(hostname)
		require.NoError(t, err, hostname)
		require.NotNil(t, domain, hostname)
	}
}

// TestNormalizeHostname pins each normalization, and the inputs it deliberately leaves alone
func TestNormalizeHostname(t *testing.T) {

	factory := &factoryCore{}

	table := []struct {
		input  string
		output string
	}{
		{"example.com", "example.com"},
		{"Example.COM", "example.com"},
		{"example.com:8080", "example.com"},
		{"www.example.com", "example.com"},
		{"WWW.Example.com", "example.com"},
		{"Www.example.com:80", "example.com"},
		{"www.www.example.com", "www.example.com"}, // only ONE leading "www." is removed
		{"wwwexample.com", "wwwexample.com"},       // "www" is a label, not a prefix
		{"sub.www.example.com", "sub.www.example.com"},
		{"localhost:8080", "localhost"},
		{"", ""},
		{":8080", ""},
	}

	for _, test := range table {
		require.Equal(t, test.output, factory.normalizeHostname(test.input), test.input)
	}
}

// TestServer covers database lookups for served and unserved hostnames
func TestServer(t *testing.T) {

	factory := testPersonalizedFactory("one.example.com")

	t.Run("Served", func(t *testing.T) {
		_, err := factory.Server("ONE.example.com:8080")
		require.NoError(t, err)
	})

	t.Run("NotServed", func(t *testing.T) {
		server, err := factory.Server("two.example.com")
		require.Error(t, err)
		require.Equal(t, http.StatusMisdirectedRequest, derp.ErrorCode(err))
		require.Nil(t, server)

		session, err := factory.Session(context.Background(), "two.example.com")
		require.Error(t, err)
		require.Equal(t, http.StatusMisdirectedRequest, derp.ErrorCode(err))
		require.Nil(t, session)
	})
}

// TestDeleteDomain covers a successful removal and a failed save
func TestDeleteDomain(t *testing.T) {

	domain := config.Domain{DomainID: "1", Hostname: "WWW.One.example.com", MasterKey: testMasterKey}

	t.Run("Success", func(t *testing.T) {

		factory := testPersonalizedFactory("one.example.com")
		factory.storage = &stubStorage{stored: configWithDomains(domain)}
		setTestConfig(factory, configWithDomains(domain))

		require.NoError(t, factory.DeleteDomain("1"))

		stored, err := factory.storage.Read()
		require.NoError(t, err)
		require.Empty(t, stored.Domains)
		require.Empty(t, factory.ListDomains())

		// The domain stops answering at once, without waiting for the saved configuration to echo back
		_, err = factory.ByHostname("one.example.com")
		require.Equal(t, http.StatusMisdirectedRequest, derp.ErrorCode(err), "a deleted domain must leave the registry")
	})

	t.Run("SaveFails", func(t *testing.T) {

		factory := testPersonalizedFactory("one.example.com")
		factory.storage = &faultyStorage{writeError: derp.Internal("test", "Storage is down")}
		setTestConfig(factory, configWithDomains(domain))

		err := factory.DeleteDomain("1")
		require.Error(t, err)
		require.Len(t, factory.ListDomains(), 1, "a failed save must leave the configuration alone")

		_, err = factory.ByHostname("one.example.com")
		require.NoError(t, err, "a failed save must leave the domain serving")
	})

	t.Run("UnknownDomain", func(t *testing.T) {

		factory := testPersonalizedFactory("one.example.com")
		factory.storage = &stubStorage{stored: configWithDomains(domain)}
		setTestConfig(factory, configWithDomains(domain))

		require.NoError(t, factory.DeleteDomain("2"))
		require.Len(t, factory.ListDomains(), 1)

		_, err := factory.ByHostname("one.example.com")
		require.NoError(t, err, "deleting another domain must not touch this one")
	})
}

// TestDeleteDomain_ClosesTheDomain verifies that deleting a live domain stops its change stream
// watchers and disconnects its database client, neither of which ends on its own.
func TestDeleteDomain_ClosesTheDomain(t *testing.T) {

	factory, domainConfig := newLiveDomainCore(t)
	factory.storage = &stubStorage{stored: configWithDomains(domainConfig)}
	setTestConfig(factory, configWithDomains(domainConfig))

	deleted := startLiveDomain(t, factory, domainConfig)
	require.Positive(t, countDomainWatchers(), "the live domain should be running watchers")

	require.NoError(t, factory.DeleteDomain(domainConfig.DomainID))

	_, err := factory.ByHostname(domainConfig.Hostname)
	require.Equal(t, http.StatusMisdirectedRequest, derp.ErrorCode(err))
	requireWatchersStopped(t, 0, "a deleted domain's watchers must stop")
	requireEventuallyDisconnected(t, deleted.Server().Client(), "a deleted domain's database client must be disconnected")
}

// TestRefreshDomain_NormalizesRegistryKey verifies that a domain configured with a hostname that
// is not already normalized is found by lookups, and refreshed in place by the next reload.
func TestRefreshDomain_NormalizesRegistryKey(t *testing.T) {

	factory, domainConfig := newLiveDomainCore(t)
	normalized := domainConfig.Hostname
	domainConfig.Hostname = "WWW." + strings.ToUpper(normalized)

	domain := startLiveDomain(t, factory, domainConfig)

	found, err := factory.ByHostname(normalized)
	require.NoError(t, err)
	require.Same(t, domain, found)

	// A second reload must refresh the same factory, not build a second one beside it
	reloadDomains(factory, configWithDomains(domainConfig))
	waitForDomainStartup(t)

	require.Equal(t, 1, factory.domains.Size())

	found, err = factory.ByHostname(normalized)
	require.NoError(t, err)
	require.Same(t, domain, found)
}

// TestRefreshDomain_RefusesCollidingHostname verifies that a domain whose hostname normalizes to
// another domain's registry key is refused, instead of taking over that domain's factory.
func TestRefreshDomain_RefusesCollidingHostname(t *testing.T) {

	factory := testPersonalizedFactory("one.example.com")
	existing, _ := factory.domains.Load("one.example.com")

	err := factory.refreshDomain(config.Domain{DomainID: "2", Hostname: "WWW.One.Example.com"})

	require.Error(t, err)
	require.Equal(t, http.StatusConflict, derp.ErrorCode(err))

	found, loaded := factory.domains.Load("one.example.com")
	require.True(t, loaded)
	require.Same(t, existing, found, "the existing domain must keep its factory")
}

// TestPutDomain_RefusesCollidingHostname verifies that the setup console cannot save a domain
// whose hostname normalizes to another domain's, while an edit to the same domain passes.
func TestPutDomain_RefusesCollidingHostname(t *testing.T) {

	existing := config.Domain{DomainID: "1", Hostname: "one.example.com"}

	t.Run("OtherDomain", func(t *testing.T) {

		factory := testPersonalizedFactory()
		storage := &faultyStorage{}
		factory.storage = storage
		setTestConfig(factory, configWithDomains(existing))
		setTestCommonDatabase(factory, lazyDatabase(t, "put-domain-collision"))

		err := factory.PutDomain(config.Domain{DomainID: "2", Hostname: "WWW.One.Example.com:8080"})

		require.Error(t, err)
		require.Equal(t, http.StatusBadRequest, derp.ErrorCode(err))
		require.Equal(t, "server.Factory.PutDomain", derp.Location(err), "returned un-wrapped, for the setup console")
		require.Zero(t, storage.writes, "a refused domain must not be saved")
	})

	t.Run("SameDomain", func(t *testing.T) {

		factory := testPersonalizedFactory()
		storage := &faultyStorage{writeError: derp.Internal("test", "Storage is down")}
		factory.storage = storage
		setTestConfig(factory, configWithDomains(existing))
		setTestCommonDatabase(factory, lazyDatabase(t, "put-domain-edit"))

		// An edit reaches the save, whose failure is the storage error rather than a refusal
		err := factory.PutDomain(config.Domain{DomainID: "1", Hostname: "One.Example.com"})

		require.Error(t, err)
		require.Equal(t, http.StatusInternalServerError, derp.ErrorCode(err))
		require.Equal(t, 1, storage.writes)
	})
}

// TestPutDomain_RequiresCommonDatabase verifies that a domain is refused, and nothing is saved,
// until the common database is connected.
func TestPutDomain_RequiresCommonDatabase(t *testing.T) {

	factory := testPersonalizedFactory()
	factory.storage = &stubStorage{}
	setTestConfig(factory, config.DefaultConfig())

	err := factory.PutDomain(config.Domain{DomainID: "1", Hostname: "one.example.com"})

	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, derp.ErrorCode(err))
	require.Equal(t, "server.Factory.PutDomain", derp.Location(err), "returned un-wrapped, for the setup console")

	stored, readErr := factory.storage.Read()
	require.NoError(t, readErr)
	require.Empty(t, stored.Domains)
}

// TestPutDomain_OmitsSecretsFromErrors verifies that a failed save does not copy the domain's
// MasterKey or database password into the error, which error reporters persist.
func TestPutDomain_OmitsSecretsFromErrors(t *testing.T) {

	factory := testPersonalizedFactory()
	factory.storage = &faultyStorage{writeError: derp.Internal("test", "Storage is down")}
	setTestConfig(factory, config.DefaultConfig())
	setTestCommonDatabase(factory, lazyDatabase(t, "put-domain-secrets"))

	err := factory.PutDomain(config.Domain{
		DomainID:      "1",
		Hostname:      "one.example.com",
		ConnectString: "mongodb://user:db-password-secret@127.0.0.1:59999/",
		MasterKey:     testMasterKey,
	})

	require.Error(t, err)
	requireNoSecret(t, err, testMasterKey)
	requireNoSecret(t, err, "db-password-secret")
}

// TestRefreshDomain_RequiresCommonDatabase verifies the guard against building a domain factory
// with no common database behind it.
func TestRefreshDomain_RequiresCommonDatabase(t *testing.T) {

	factory := testPersonalizedFactory()

	err := factory.refreshDomain(config.Domain{DomainID: "1", Hostname: "one.example.com"})

	require.Error(t, err)
	require.Equal(t, http.StatusInternalServerError, derp.ErrorCode(err))
	require.Zero(t, factory.domains.Size())
}

// TestRefreshDomains_UnreachableDomainIsLeftOut verifies that a new domain that cannot load is
// absent from the registry, so requests to it answer 421.
func TestRefreshDomains_UnreachableDomainIsLeftOut(t *testing.T) {

	recordReports(t)

	factory := testPersonalizedFactory()
	setTestFilesystems(factory)
	setTestCommonDatabase(factory, lazyDatabase(t, "unreachable-domain"))

	reloadDomains(factory, configWithDomains(config.Domain{
		DomainID:      "1",
		Hostname:      "unreachable.example.com",
		ConnectString: unreachableConnection,
		DatabaseName:  "unreachable",
	}))

	_, err := factory.ByHostname("unreachable.example.com")
	require.Error(t, err)
	require.Equal(t, http.StatusMisdirectedRequest, derp.ErrorCode(err))
}

/******************************************
 * Domain Registry (MongoDB integration)
 ******************************************/

// TestRefreshDomains_Lifecycle creates, refreshes, and removes a live domain factory.  A removed
// factory's watchers must stop, because they reopen themselves until canceled (BUG-170).
func TestRefreshDomains_Lifecycle(t *testing.T) {

	factory, domainConfig := newLiveDomainCore(t)
	baseline := countDomainWatchers()

	// A new domain gets a factory, and that factory starts its watchers
	created := startLiveDomain(t, factory, domainConfig)

	running := countDomainWatchers()
	require.Greater(t, running, baseline, "a new domain factory must start its watchers")

	// A reload that keeps the domain refreshes the SAME factory, and leaves its watchers alone
	reloadDomains(factory, configWithDomains(domainConfig))

	kept, err := factory.ByHostname(domainConfig.Hostname)
	require.NoError(t, err)
	require.Same(t, created, kept)
	require.False(t, kept.MarkForDeletion)
	require.Equal(t, running, countDomainWatchers())

	// A reload that omits the domain removes it, and closes it
	reloadDomains(factory, config.DefaultConfig())

	_, err = factory.ByHostname(domainConfig.Hostname)
	require.Error(t, err)
	require.Zero(t, factory.domains.Size())
	requireWatchersStopped(t, baseline, "a removed domain's watchers must stop")
	requireEventuallyDisconnected(t, created.Server().Client(), "a removed domain's database client must be disconnected")
}

// TestRefreshDomain_ReconnectClosesPreviousClient verifies that moving a live domain to another
// database disconnects the client it used before, and restarts its watchers on the new one.
func TestRefreshDomain_ReconnectClosesPreviousClient(t *testing.T) {

	factory, domainConfig := newLiveDomainCore(t)
	created := startLiveDomain(t, factory, domainConfig)
	previous := created.Server().Client()
	running := countDomainWatchers()

	// Move the same domain to a new database
	moved := domainConfig
	moved.DatabaseName = domainConfig.DatabaseName + "_moved"

	reloadDomains(factory, configWithDomains(moved))
	waitForDomainStartup(t)

	kept, err := factory.ByHostname(domainConfig.Hostname)
	require.NoError(t, err)
	require.Same(t, created, kept, "a reconnect refreshes the same factory")
	require.NotSame(t, previous, kept.Server().Client())

	requireEventuallyDisconnected(t, previous, "the client a reconnect replaced must be disconnected")
	requireWatchersStopped(t, running, "the old watchers must stop, and exactly as many must start again")
}

// TestRefreshDomains_FailedNewDomainReleasesItself verifies that a new domain which fails to load
// stops the realtime broker it started, rather than leaking one on every reload that retries it.
func TestRefreshDomains_FailedNewDomainReleasesItself(t *testing.T) {

	recordReports(t)

	factory := testPersonalizedFactory()
	setTestFilesystems(factory)
	setTestCommonDatabase(factory, lazyDatabase(t, "failed-new-domain"))

	brokers := countGoroutinesStartedBy("realtime.NewBroker")

	reloadDomains(factory, configWithDomains(config.Domain{
		DomainID:      "1",
		Hostname:      "unreachable.example.com",
		ConnectString: unreachableConnection,
		DatabaseName:  "unreachable",
	}))

	require.Zero(t, factory.domains.Size())
	require.Eventually(t, func() bool { return countGoroutinesStartedBy("realtime.NewBroker") == brokers }, 5*time.Second, 20*time.Millisecond, "a domain that failed to load must stop its broker")
}

// TestRemoveAllDomains verifies that every domain factory leaves the registry and is closed
func TestRemoveAllDomains(t *testing.T) {

	factory, domainConfig := newLiveDomainCore(t)
	created := startLiveDomain(t, factory, domainConfig)

	factory.reloadLock.Lock()
	factory.removeAllDomains()
	factory.reloadLock.Unlock()

	require.Zero(t, factory.domains.Size())
	requireWatchersStopped(t, 0, "every removed domain's watchers must stop")
	requireEventuallyDisconnected(t, created.Server().Client(), "every removed domain's database client must be disconnected")

	// An empty registry is not an error
	require.NotPanics(t, factory.removeAllDomains)
}

// TestRefreshCommonDatabase_VerifyFailureClosesDomains verifies that when a new common database
// fails its ping, every domain factory is dropped AND closed, not just dropped.
func TestRefreshCommonDatabase_VerifyFailureClosesDomains(t *testing.T) {

	factory, domainConfig := newLiveDomainCore(t)
	created := startLiveDomain(t, factory, domainConfig)

	_, err := verifyCommonDatabase(factory, lifecycleConnection("59999", "verify-failure"))
	require.Error(t, err)

	require.Zero(t, factory.domains.Size())
	requireWatchersStopped(t, 0, "a dropped domain's watchers must stop")
	requireEventuallyDisconnected(t, created.Server().Client(), "a dropped domain's database client must be disconnected")
}

// TestSetupFactory_UpdateConfig_ClosesReplacedDomains verifies that when the setup console saves a
// new common database, the domain factories it rebuilds are closed rather than abandoned.
func TestSetupFactory_UpdateConfig_ClosesReplacedDomains(t *testing.T) {

	factory, domainConfig := newLiveSetupFactory(t)
	storage := &stubStorage{stored: configWithDomains(domainConfig)}
	factory.storage = storage
	setTestConfig(&factory.factoryCore, storage.stored)

	replaced := startLiveDomain(t, &factory.factoryCore, domainConfig)
	running := countDomainWatchers()

	// Save the same configuration, pointing at a different common database
	edited := factory.Config()
	edited.ActivityPubCache = mapof.String{
		"connectString": liveTestConnection,
		"database":      "emissary_servertest_common_" + domainConfig.DomainID + "_moved",
	}

	require.NoError(t, factory.UpdateConfig(edited))

	t.Cleanup(func() {
		disconnectCommonDatabase(factory.CommonDatabase())
	})

	// The domain is rebuilt as a new factory, and the old one is closed
	rebuilt, err := factory.ByHostname(domainConfig.Hostname)
	require.NoError(t, err)
	releaseDomainFactory(t, rebuilt)
	waitForDomainStartup(t)

	require.NotSame(t, replaced, rebuilt)
	requireWatchersStopped(t, running, "the replaced domain's watchers must stop")
	requireEventuallyDisconnected(t, replaced.Server().Client(), "the replaced domain's database client must be disconnected")
}

// TestRefreshDomains_FailedRefreshKeepsDomain verifies that an existing domain whose refresh
// fails stays in the registry, rather than being removed with the domains that left the config.
func TestRefreshDomains_FailedRefreshKeepsDomain(t *testing.T) {

	factory, domainConfig := newLiveDomainCore(t)
	created := startLiveDomain(t, factory, domainConfig)

	// Point the same domain at a server that cannot answer
	recordReports(t)

	broken := domainConfig
	broken.ConnectString = unreachableConnection

	reloadDomains(factory, configWithDomains(broken))

	kept, err := factory.ByHostname(domainConfig.Hostname)
	require.NoError(t, err)
	require.Same(t, created, kept)
}

// TestPutDomain_Live adds a live domain, then saves it again with a valid and an invalid owner
func TestPutDomain_Live(t *testing.T) {

	factory, domainConfig := newLiveDomainCore(t)
	setTestConfig(factory, config.DefaultConfig())

	// A domain with no owner is saved, and gets a factory
	require.NoError(t, factory.PutDomain(domainConfig))

	domain, err := factory.ByHostname(domainConfig.Hostname)
	require.NoError(t, err)
	releaseDomainFactory(t, domain)
	waitForDomainStartup(t)

	stored, err := factory.storage.Read()
	require.NoError(t, err)

	storedDomain, found := stored.Domains.Get(domainConfig.DomainID)
	require.True(t, found)
	require.Equal(t, domainConfig.MasterKey, storedDomain.MasterKey)
	require.Len(t, factory.ListDomains(), 1)

	// Saved again with an owner, the same factory writes the owner into its own database
	domainConfig.Owner = config.Owner{DisplayName: "Alice", Username: "alice", EmailAddress: "alice@example.com"}
	require.NoError(t, factory.PutDomain(domainConfig))

	refreshed, err := factory.ByHostname(domainConfig.Hostname)
	require.NoError(t, err)
	require.Same(t, domain, refreshed)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := domain.Server().Database().Collection("User").CountDocuments(ctx, bson.M{"username": "alice", "isOwner": true})
	require.NoError(t, err)
	require.Equal(t, int64(1), count)

	// An owner the User service rejects fails the save ("owner" is not a valid username)
	domainConfig.Owner = config.Owner{DisplayName: "Owner", Username: "owner", EmailAddress: "owner@example.com"}
	err = factory.PutDomain(domainConfig)
	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, derp.ErrorCode(err))

	// The same session machinery serves requests
	session, err := factory.Session(ctx, domainConfig.Hostname)
	require.NoError(t, err)
	session.Close()
}

// TestPutDomain_UnreachableDomain verifies that a domain whose database cannot load is saved but
// never served, and that the save is not rolled back.
func TestPutDomain_UnreachableDomain(t *testing.T) {

	recordReports(t)

	factory := testPersonalizedFactory()
	factory.storage = &stubStorage{}
	setTestConfig(factory, config.DefaultConfig())
	setTestFilesystems(factory)
	setTestCommonDatabase(factory, lazyDatabase(t, "put-domain-unreachable"))

	err := factory.PutDomain(config.Domain{
		DomainID:      "1",
		Hostname:      "unreachable.example.com",
		ConnectString: unreachableConnection,
		DatabaseName:  "unreachable",
		MasterKey:     testMasterKey,
	})

	require.Error(t, err)
	require.Zero(t, factory.domains.Size())
	require.Len(t, factory.ListDomains(), 1)
}

// Two paths in factory_core.go have no test.  PutDomain's ByHostname failure needs a configured
// hostname that is not already normalized, or a reload that removes the domain mid-call; and
// Session's error branch is unreachable, because data-mongo's Server.Session never fails.

// TestTestConnection_Live verifies that a reachable database passes the setup console's check
func TestTestConnection_Live(t *testing.T) {

	requireLiveMongo(t)

	factory := &factoryCore{}

	require.NoError(t, factory.TestConnection(config.Domain{
		ConnectString: liveTestConnection,
		DatabaseName:  "emissary_servertest_connection",
	}))
}

/******************************************
 * Client Addresses and Ports
 ******************************************/

// TestCalcClientIPStrategy covers each configured strategy, and the fallback for bad settings
func TestCalcClientIPStrategy(t *testing.T) {

	recorder := recordReports(t)
	factory := &factoryCore{}

	// strategy returns the strategy built from these settings
	strategy := func(name string, header string, count int) realclientip.Strategy {
		value := config.DefaultConfig()
		value.ClientIPStrategy = name
		value.ClientIPHeader = header
		value.ClientIPTrustedCount = count
		return factory.calcClientIPStrategy(value)
	}

	// Valid settings build the named strategy, and report nothing
	require.IsType(t, realclientip.RemoteAddrStrategy{}, strategy("", "", 0))
	require.IsType(t, realclientip.RemoteAddrStrategy{}, strategy("REMOTE-ADDR", "", 0))
	require.IsType(t, realclientip.RightmostTrustedCountStrategy{}, strategy("RIGHTMOST-TRUSTED-COUNT", "", 1))
	require.IsType(t, realclientip.SingleIPHeaderStrategy{}, strategy("SINGLE-IP-HEADER", "X-Real-IP", 0))
	require.Empty(t, recorder.reported())

	// Invalid settings fall back to REMOTE-ADDR, and report why
	require.IsType(t, realclientip.RemoteAddrStrategy{}, strategy("RIGHTMOST-TRUSTED-COUNT", "", 0))
	require.IsType(t, realclientip.RemoteAddrStrategy{}, strategy("SINGLE-IP-HEADER", "", 0))
	require.IsType(t, realclientip.RemoteAddrStrategy{}, strategy("remote-addr", "", 0)) // names are case-sensitive
	require.IsType(t, realclientip.RemoteAddrStrategy{}, strategy("UNKNOWN", "", 0))
	require.Len(t, recorder.reported(), 4)
}

// TestClientIP reads the client address through the published strategy
func TestClientIP(t *testing.T) {

	request := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	request.RemoteAddr = "192.0.2.10:4567"
	request.Header.Set("X-Real-IP", "198.51.100.20")

	factory := &factoryCore{}

	factory.rewire(func(value *wiring) {
		value.clientIPStrategy = realclientip.RemoteAddrStrategy{}
	})

	require.Equal(t, "192.0.2.10", factory.ClientIP(request))

	// A strategy swap is picked up by the very next request
	headerStrategy, err := realclientip.NewSingleIPHeaderStrategy("X-Real-IP")
	require.NoError(t, err)

	factory.rewire(func(value *wiring) {
		value.clientIPStrategy = headerStrategy
	})

	require.Equal(t, "198.51.100.20", factory.ClientIP(request))

	// With no strategy, there is no address to report
	recorder := recordReports(t)
	empty := &factoryCore{}

	require.Empty(t, empty.ClientIP(request))
	require.Len(t, recorder.reported(), 1)
}

// TestPort verifies that only local domains carry a port, and only when it is not 80
func TestPort(t *testing.T) {

	factory := &factoryCore{}

	// port returns the suffix for this hostname under this HTTP port
	port := func(hostname string, httpPort int) string {
		value := config.DefaultConfig()
		value.HTTPPort = httpPort
		setTestConfig(factory, value)
		return factory.port(config.Domain{Hostname: hostname})
	}

	require.Empty(t, port("example.com", 8080))
	require.Empty(t, port("localhost", 0))
	require.Empty(t, port("localhost", 80))
	require.Equal(t, ":8080", port("localhost", 8080))
	require.Equal(t, ":8080", port("emissary.local", 8080))
}

/******************************************
 * Shared Lifecycle Steps
 ******************************************/

// TestSetLogLevel maps every configured level, and falls back to Info for anything unknown
func TestSetLogLevel(t *testing.T) {

	previous := zerolog.GlobalLevel()
	t.Cleanup(func() { zerolog.SetGlobalLevel(previous) })

	table := []struct {
		name  string
		level zerolog.Level
	}{
		{"Trace", zerolog.TraceLevel},
		{"Debug", zerolog.DebugLevel},
		{"Info", zerolog.InfoLevel},
		{"Warn", zerolog.WarnLevel},
		{"Error", zerolog.ErrorLevel},
		{"Fatal", zerolog.FatalLevel},
		{"Panic", zerolog.PanicLevel},
		{"None", zerolog.Disabled},
		{"", zerolog.InfoLevel},
		{"debug", zerolog.InfoLevel}, // names are case-sensitive
		{"Verbose", zerolog.InfoLevel},
	}

	for _, test := range table {
		value := config.DefaultConfig()
		value.DebugLevel = test.name
		setLogLevel(value)
		require.Equal(t, test.level, zerolog.GlobalLevel(), test.name)
	}
}
