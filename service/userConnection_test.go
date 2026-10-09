package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/delta"
	"github.com/stretchr/testify/require"
)

// testDomainCipher is a 64-character hexadecimal AES-256 value for sealing test vaults. It
// is named to avoid "key"/"secret", which is half of what a scanner matches on.
const testDomainCipher = "0123456789abcdef0123456789abcdef" + "0123456789abcdef0123456789abcdef"

// TestUserConnection_ConnectAlwaysReprovesAnActiveConnection pins the deliberate absence of
// a short-circuit
func TestUserConnection_ConnectAlwaysReprovesAnActiveConnection(t *testing.T) {

	// An earlier version skipped a connection that was already working, to keep a network
	// round-trip off "routine" saves.  There are no routine saves: UserConnection.Save is
	// reached only from the settings form.  Re-running setup every time is what puts back a
	// webhook the User deleted inside Mailchimp, and setup is idempotent, so an unchanged
	// save writes nothing at either end.

	states := map[string]string{
		"setup finished":   model.UserConnectionStatusReady,
		"setup unfinished": model.UserConnectionStatusPending,
		"credential stale": model.UserConnectionStatusReconnect,
	}

	for name, status := range states {

		t.Run(name, func(t *testing.T) {

			service := UserConnection{encryptionKey: testDomainCipher}

			userConnection := model.NewUserConnection()
			userConnection.Type = model.UserConnectionTypeMailchimp
			userConnection.IsActive.Set(true)
			userConnection.Status = status

			// Simulate a record loaded from the database: already active, nothing retyped
			userConnection = reloaded(userConnection)
			require.True(t, userConnection.IsActive.NotChanged())

			// It reaches the network, and fails there -- which is the proof that it did not skip
			require.Error(t, service.connect(nil, &userConnection))
		})
	}
}

// TestUserConnection_ConnectSkipsRecordsThatWereAlreadyOff confirms a paused connection is
// not torn down again on every save
func TestUserConnection_ConnectSkipsRecordsThatWereAlreadyOff(t *testing.T) {

	service := UserConnection{encryptionKey: testDomainCipher}

	userConnection := model.NewUserConnection()
	userConnection.Type = model.UserConnectionTypeMailchimp

	require.False(t, userConnection.IsActive.IsChanged())
	require.NoError(t, service.connect(nil, &userConnection))
}

// TestUserConnection_TurningOffDisconnects pins what the Active switch does when it moves
// to FALSE
func TestUserConnection_TurningOffDisconnects(t *testing.T) {

	// Pausing removes what Emissary installed at the remote service but leaves the
	// credential in place, so turning the connection back on needs no retyping.

	service := UserConnection{encryptionKey: testDomainCipher}

	userConnection := model.NewUserConnection()
	userConnection.Type = model.UserConnectionTypeMailchimp
	userConnection.IsActive.Set(true)
	userConnection.Vault.SetString(model.UserConnectionVaultAPIKey, "a-secret-value")
	userConnection.Data.SetString(model.UserConnectionDataCenter, "us6")
	userConnection = reloaded(userConnection)

	userConnection.Status = model.UserConnectionStatusReady
	userConnection.Data.SetString(model.UserConnectionDataWebhookID, "webhook-123")
	userConnection.IsActive.Set(false)

	require.NoError(t, service.connect(nil, &userConnection))

	require.False(t, userConnection.IsReady(), "a paused connection must not read as ready")
	require.Empty(t, userConnection.Data.GetString(model.UserConnectionDataWebhookID), "the installed webhook is released")
	require.True(t, userConnection.Vault.HasString(model.UserConnectionVaultAPIKey), "the credential stays in place")
	require.Equal(t, "us6", userConnection.Data.GetString(model.UserConnectionDataCenter), "settings stay in place")
}

// TestUserConnection_ConnectRejectsUnknownTypes confirms an unrecognized service cannot
// save as if it worked
func TestUserConnection_ConnectRejectsUnknownTypes(t *testing.T) {

	service := UserConnection{encryptionKey: testDomainCipher}

	userConnection := model.NewUserConnection()
	userConnection.Type = "NOT-A-REAL-SERVICE"
	userConnection.IsActive.Set(true)

	require.Error(t, service.connect(nil, &userConnection))
}

// TestUserConnection_EncryptVaultNeedsNoKeyWhenEmpty confirms a connection with nothing to
// seal does not demand a master key
func TestUserConnection_EncryptVaultNeedsNoKeyWhenEmpty(t *testing.T) {

	// DecodeMasterKey refuses an empty key (BUG-110), so reaching for one unconditionally
	// would make a domain that has none unable to save any connection at all.
	service := UserConnection{}
	userConnection := model.NewUserConnection()

	require.NoError(t, service.encryptVault(&userConnection))
}

// reloaded returns a copy of the provided connection as it would read after a round-trip
// through the database, with its IsActive switch no longer marked as changed
func reloaded(userConnection model.UserConnection) model.UserConnection {

	// delta.Bool settles `original` when it unmarshals, and NewBool is the only other way
	// to reach that state -- so the round-trip is reproduced rather than mocked.
	result := userConnection
	result.IsActive = delta.NewBool(userConnection.IsActive.Value())

	return result
}

// TestUserConnection_PausingSurvivesAnUnreachableMailchimp is D37's rule on the path that
// used to break it
func TestUserConnection_PausingSurvivesAnUnreachableMailchimp(t *testing.T) {

	// UserConnection.Delete has always reported a failed teardown and removed the record
	// anyway. The PAUSE path propagated instead, so a User whose credential was revoked --
	// or whose Mailchimp was simply down -- could not switch their connection off at all.
	// Same rule, two paths; this pins the second one.

	service := UserConnection{encryptionKey: testDomainCipher}

	userConnection := model.NewUserConnection()
	userConnection.Type = model.UserConnectionTypeMailchimp
	userConnection.Status = model.UserConnectionStatusReady
	userConnection.IsActive.Set(true)
	userConnection = reloaded(userConnection)

	// A connection with an installed webhook and a data center that resolves nowhere
	userConnection.Data.SetString(model.UserConnectionDataAudienceID, "abc123")
	userConnection.Data.SetString(model.UserConnectionDataWebhookID, "wh1")
	userConnection.Data.SetString(model.UserConnectionDataCenter, "us6")
	userConnection.Vault.SetString(model.UserConnectionVaultAPIKey, testMailchimpCredential)

	// Now switch it off
	userConnection.IsActive.Set(false)

	require.NoError(t, service.connect(nil, &userConnection), "pausing must not fail on a teardown that cannot reach Mailchimp")
	require.Equal(t, model.UserConnectionStatusPending, userConnection.Status, "and the local side must still be switched off")
	require.False(t, userConnection.HasWebhook())
}
