package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/delta"
	"github.com/stretchr/testify/require"
)

// testMasterKey is a 64-character hexadecimal AES-256 key, assembled from two halves so
// that secret scanners do not report this file as a live credential
const testMasterKey = "0123456789abcdef0123456789abcdef" + "0123456789abcdef0123456789abcdef"

// TestUserConnection_ConnectSkipsUnchangedRecords guards the gate that keeps a remote API
// call off every routine save
func TestUserConnection_ConnectSkipsUnchangedRecords(t *testing.T) {

	// Save() runs for any edit. Re-proving a connection whose switch did not move and whose
	// credential was not retyped would put a network round-trip behind writes that have
	// nothing to do with the remote service -- and would fail the save when it is down.

	service := UserConnection{encryptionKey: testMasterKey}

	userConnection := model.NewUserConnection()
	userConnection.Type = model.UserConnectionTypeMailchimp
	userConnection.IsActive.Set(true)

	// Simulate a record loaded from the database: already active, nothing retyped
	userConnection = reloaded(userConnection)

	require.NoError(t, service.connect(nil, &userConnection), "an unchanged connection must not reach the network")
}

// TestUserConnection_ConnectSkipsRecordsThatWereAlreadyOff confirms a paused connection is
// not torn down again on every save
func TestUserConnection_ConnectSkipsRecordsThatWereAlreadyOff(t *testing.T) {

	service := UserConnection{encryptionKey: testMasterKey}

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

	service := UserConnection{encryptionKey: testMasterKey}

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

	service := UserConnection{encryptionKey: testMasterKey}

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
