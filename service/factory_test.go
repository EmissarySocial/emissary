package service

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/realtime"
	mongodb "github.com/benpate/data-mongo"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"golang.org/x/crypto/bcrypt"
)

/******************************************
 * Password Policy
 *
 * The password chain (primary selection, rehash flagging, truncation,
 * fuzz) is tested in steranko, where the PasswordService lives.  These
 * tests pin the POLICY that Factory.Steranko configures, through the
 * real factory wiring, so a config change cannot slip through unseen.
 * Hashing never touches the database, so a zero Factory and a nil
 * session are safe.
 ******************************************/

// TestSteranko_SetPassword_StoresBCrypt12 pins that a User can only ever receive a bcrypt hash at
// cost 12, never a plaintext password (CWE-256; see AGENTS.md).
func TestSteranko_SetPassword_StoresBCrypt12(t *testing.T) {

	factory := Factory{}
	steranko := factory.Steranko(nil)

	user := model.NewUser()
	require.Nil(t, steranko.SetPassword(&user, "TestPass123!"))

	require.NotEqual(t, "TestPass123!", user.Password)
	require.Nil(t, bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("TestPass123!")))

	cost, err := bcrypt.Cost([]byte(user.Password))
	require.Nil(t, err, "stored password must be a bcrypt hash: %q", user.Password)
	require.Equal(t, 12, cost)
}

// TestSteranko_PlaintextFallback pins that a legacy plaintext password still verifies, and is
// flagged for re-hashing, so that pre-hashing accounts can sign in.
func TestSteranko_PlaintextFallback(t *testing.T) {

	// When the plaintext-password migration ships, the fallback leaves Factory.Steranko and this
	// test must be DELETED deliberately.
	factory := Factory{}
	steranko := factory.Steranko(nil)

	ok, rehash := steranko.ComparePassword("legacy-password", "legacy-password")
	require.True(t, ok)
	require.True(t, rehash, "plaintext matches must be flagged for upgrade")
}

/******************************************
 * Lifecycle
 *
 * Refresh itself needs a fully wired server factory and a replica set,
 * so its WatchDomain wiring is untested here; queries tests WatchDomain.
 ******************************************/

// TestShouldStartDomainService pins when Refresh restarts the Domain service: on a rename as well
// as a reconnect, but never before a database exists.
func TestShouldStartDomainService(t *testing.T) {

	configured := config.Domain{
		ConnectString: "mongodb://localhost:27017",
		DatabaseName:  "emissary_localhost",
	}

	testCases := []struct {
		name               string
		newConfig          config.Domain
		hasDatabaseChanged bool
		hasHostnameChanged bool
		expected           bool
	}{
		{"NothingChanged", configured, false, false, false},
		{"DatabaseChanged", configured, true, false, true},
		{"HostnameChanged", configured, false, true, true},
		{"BothChanged", configured, true, true, true},

		// Setup mode: the domain is renamed before its database is configured, so there is
		// nothing to open a session on and Start must be skipped.
		{"HostnameChangedWithNoConnectString", config.Domain{DatabaseName: "emissary"}, false, true, false},
		{"HostnameChangedWithNoDatabaseName", config.Domain{ConnectString: "mongodb://localhost:27017"}, false, true, false},
		{"HostnameChangedWithNoDatabaseAtAll", config.Domain{}, false, true, false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := shouldStartDomainService(testCase.newConfig, testCase.hasDatabaseChanged, testCase.hasHostnameChanged)
			require.Equal(t, testCase.expected, result)
		})
	}
}

// TestFactory_StopWatchers pins that the change stream watchers a factory starts can be stopped,
// since they never stop on their own.
func TestFactory_StopWatchers(t *testing.T) {

	t.Run("ZeroFactory", func(t *testing.T) {
		factory := Factory{}
		require.NotPanics(t, factory.StopWatchers)
	})

	t.Run("CancelsTheContext", func(t *testing.T) {
		factory := Factory{}
		ctx := factory.newRefreshContext()

		factory.StopWatchers()

		require.Error(t, ctx.Err())
		require.NotPanics(t, factory.StopWatchers)
	})

	t.Run("NewContextStopsTheOldOne", func(t *testing.T) {
		factory := Factory{}
		first := factory.newRefreshContext()
		second := factory.newRefreshContext()

		require.Error(t, first.Err())
		require.Nil(t, second.Err())
	})
}

// newClosableFactory returns a factory holding everything Close releases: a broker, a lazy database
// client that never contacts a server, and a shared JWT service whose cache is already in use.
func newClosableFactory(t *testing.T) *Factory {

	t.Helper()

	jwtService := NewJWT()
	jwtService.Refresh(nil)
	jwtService.cache.Set("shared-key", []byte("shared-value"))

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(lazyConnection))
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = client.Disconnect(context.Background()) // Close may already have disconnected it
	})

	factory := &Factory{
		jwtService:       &jwtService,
		server:           mongodb.NewServer(client.Database("closable")),
		sseUpdateChannel: make(chan realtime.Message, 1),
	}

	factory.realtimeBroker = realtime.NewBroker(factory.sseUpdateChannel)
	return factory
}

// lazyConnection names a closed port.  mongo.Connect never contacts it, and Ping on a
// disconnected client fails before server selection, so no test here needs a MongoDB server.
const lazyConnection = "mongodb://127.0.0.1:59996/?directConnection=true&serverSelectionTimeoutMS=200"

// countGoroutinesStartedBy returns how many running goroutines the named function started
func countGoroutinesStartedBy(function string) int {

	// Grow the buffer until the dump of every goroutine fits
	for buffer := make([]byte, 1<<20); ; buffer = make([]byte, 2*len(buffer)) {
		if size := runtime.Stack(buffer, true); size < len(buffer) {
			return strings.Count(string(buffer[:size]), "created by github.com/EmissarySocial/emissary/"+function+" ")
		}
	}
}

// requireEventuallyDisconnected waits for the client to be disconnected, which Close does in the background
func requireEventuallyDisconnected(t *testing.T, client *mongo.Client) {
	t.Helper()
	require.Eventually(t, func() bool {
		return errors.Is(client.Ping(context.Background(), readpref.Primary()), mongo.ErrClientDisconnected)
	}, 5*time.Second, 10*time.Millisecond, "the database client was never disconnected")
}

// TestFactory_Close pins that Close releases everything the factory owns, and nothing it shares
func TestFactory_Close(t *testing.T) {

	t.Run("StopsTheWatchers", func(t *testing.T) {
		factory := newClosableFactory(t)
		watchers := factory.newRefreshContext()

		factory.Close()

		require.Error(t, watchers.Err())
	})

	t.Run("StopsTheBroker", func(t *testing.T) {
		baseline := countGoroutinesStartedBy("realtime.NewBroker")
		factory := newClosableFactory(t)
		require.Equal(t, baseline+1, countGoroutinesStartedBy("realtime.NewBroker"))

		factory.Close()

		require.Eventually(t, func() bool { return countGoroutinesStartedBy("realtime.NewBroker") == baseline }, 5*time.Second, 10*time.Millisecond)
	})

	t.Run("DisconnectsTheDatabase", func(t *testing.T) {
		factory := newClosableFactory(t)

		factory.Close()

		requireEventuallyDisconnected(t, factory.server.Client())
	})

	t.Run("LeavesTheSSEChannelOpen", func(t *testing.T) {
		factory := newClosableFactory(t)

		factory.Close()

		// A request still in progress may send after Close, and must not panic
		require.NotPanics(t, func() { factory.sseUpdateChannel <- realtime.NewMessage_Updated(primitive.NewObjectID()) })
	})

	t.Run("LeavesTheSharedJWTServiceOpen", func(t *testing.T) {
		factory := newClosableFactory(t)

		factory.Close()

		value, found := factory.jwtService.cache.Get("shared-key")
		require.True(t, found, "closing one domain must not clear the JWT cache every domain shares")
		require.Equal(t, []byte("shared-value"), value)
	})

	t.Run("IsSafeToCallTwice", func(t *testing.T) {
		factory := newClosableFactory(t)

		factory.Close()

		require.NotPanics(t, factory.Close)
	})

	t.Run("ZeroFactory", func(t *testing.T) {
		factory := Factory{}
		require.NotPanics(t, factory.Close)
	})
}

// TestDisconnectDatabase pins that a factory that never connected closes without error
func TestDisconnectDatabase(t *testing.T) {

	t.Run("NeverConnected", func(t *testing.T) {
		require.NotPanics(t, func() { disconnectDatabase(mongodb.Server{}) })
	})

	t.Run("Connected", func(t *testing.T) {
		client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(lazyConnection))
		require.NoError(t, err)

		disconnectDatabase(mongodb.NewServer(client.Database("disconnect")))

		require.ErrorIs(t, client.Ping(context.Background(), readpref.Primary()), mongo.ErrClientDisconnected)
	})
}
