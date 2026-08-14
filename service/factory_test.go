package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// Password chain mechanics (primary selection, rehash flagging, truncation, fuzz) are
// tested in the steranko package, where the PasswordService lives.  These tests pin
// the POLICY that Factory.Steranko configures — exercised through the real factory
// wiring, so a config change here cannot slip through unnoticed.  Hashing never
// touches the database, so a zero Factory and a nil session are safe.

// TestSteranko_SetPassword_StoresBCrypt12 is the regression test for the plaintext
// password vulnerability (CWE-256): registration, password reset, and Mastodon signup
// once stored raw passwords because they bypassed the hasher.  All password writes now
// route through the Steranko instance built here, so this pins that a User can only
// ever receive a bcrypt hash — at cost 12, the deliberate balance between
// offline-cracking resistance and signin latency / failed-signin CPU cost.
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

// TestSteranko_PlaintextFallback pins that legacy plaintext-stored passwords still
// verify — flagged for re-hashing — so pre-hashing accounts can sign in.  When the
// plaintext-password migration ships, the fallback is removed from Factory.Steranko
// and this test must be DELETED deliberately.
func TestSteranko_PlaintextFallback(t *testing.T) {

	factory := Factory{}
	steranko := factory.Steranko(nil)

	ok, rehash := steranko.ComparePassword("legacy-password", "legacy-password")
	require.True(t, ok)
	require.True(t, rehash, "plaintext matches must be flagged for upgrade")
}

// TestShouldStartDomainService verifies when Factory.Refresh restarts the Domain service.  Start
// reloads the stored Domain record and stamps the configured hostname into it, so it has to run on
// a rename as well as a reconnect -- but never before a database exists to run against, because a
// factory built in setup mode has no connection to open a session on.
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
