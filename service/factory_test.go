package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/realtime"
	"github.com/stretchr/testify/require"
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

// TestFactory_Close pins that Close stops the change stream watchers and closes the SSE channel
func TestFactory_Close(t *testing.T) {

	jwtService := NewJWT()
	jwtService.Refresh(nil)

	factory := Factory{
		jwtService:       &jwtService,
		sseUpdateChannel: make(chan realtime.Message),
	}

	factory.realtimeBroker = realtime.NewBroker(factory.sseUpdateChannel)
	watchers := factory.newRefreshContext()

	factory.Close()

	require.Error(t, watchers.Err())

	_, isOpen := <-factory.sseUpdateChannel
	require.False(t, isOpen)
}
