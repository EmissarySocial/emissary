package model

import (
	"encoding/json"
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

// TestUserConnection verifies that every UserConnection property round-trips through the schema
func TestUserConnection(t *testing.T) {

	userConnection := NewUserConnection()

	s := schema.New(UserConnectionSchema())

	table := []tableTestItem{
		{"userConnectionId", "123412341234123412341234", nil},
		{"userId", "123456781234567812345678", nil},
		{"type", UserConnectionTypeMailchimp, nil},
		{"status", UserConnectionStatusReady, nil},
		{"isActive", true, nil},
		{"data." + UserConnectionDataCenter, "us6", nil},
		{"data." + UserConnectionDataAudienceID, "abc123", nil},
		{"vault." + UserConnectionVaultAPIKey, "a-secret-value", VaultObscuredValue},
	}

	tableTest_Schema(t, &s, &userConnection, table)
}

// TestUserConnection_IsActiveTracksChanges pins the switch that drives connect and disconnect
func TestUserConnection_IsActiveTracksChanges(t *testing.T) {

	// The service keys on IsChanged() to decide whether to install or remove this connection
	// at the remote service. Writing the field directly instead of through the schema would
	// lose the change, and the connection would never be installed.
	userConnection := NewUserConnection()
	s := schema.New(UserConnectionSchema())

	require.False(t, userConnection.IsActive.IsChanged())

	require.NoError(t, s.Set(&userConnection, "isActive", true))

	require.True(t, userConnection.IsActive.IsChanged())
	require.True(t, userConnection.IsActive.IsTrue())
}

// TestUserConnection_SecretsDoNotExport confirms a connection's credential stays out of JSON
func TestUserConnection_SecretsDoNotExport(t *testing.T) {

	// Placement is the whole of the export policy: Vault carries `json:"-"` on every
	// persisted field, and Data does not, so a secret written to Data would ship.
	userConnection := NewUserConnection()
	userConnection.Vault.SetString(UserConnectionVaultAPIKey, "a-secret-value")
	userConnection.Vault.SetString(UserConnectionVaultWebhookSecret, "another-secret-value")
	userConnection.Data.SetString(UserConnectionDataCenter, "us6")

	exported, err := json.Marshal(userConnection)

	require.NoError(t, err)
	require.NotContains(t, string(exported), "a-secret-value")
	require.NotContains(t, string(exported), "another-secret-value")
}

// TestUserConnection_IsReady pins the single predicate that every sync hook consults
func TestUserConnection_IsReady(t *testing.T) {

	table := []struct {
		name     string
		isActive bool
		status   string
		expected bool
	}{
		{"switched on and set up", true, UserConnectionStatusReady, true},
		{"switched on, setup unfinished", true, "", false},
		{"switched on but rejected", true, UserConnectionStatusReconnect, false},
		{"switched off", false, UserConnectionStatusReady, false},
		{"switched off and rejected", false, UserConnectionStatusReconnect, false},
	}

	// The second row is the one that changed with D39.  A connection whose API key works but
	// whose audience was never chosen carries NO status, and used to pass this predicate --
	// so the inbound webhook trusted a connection that could not push or receive anything.

	for _, test := range table {

		t.Run(test.name, func(t *testing.T) {

			userConnection := NewUserConnection()
			userConnection.IsActive.Set(test.isActive)
			userConnection.Status = test.status

			require.Equal(t, test.expected, userConnection.IsReady())
		})
	}
}

// TestUserConnection_SetupStates confirms the three states a connection moves through
func TestUserConnection_SetupStates(t *testing.T) {

	// NeedsSetup and IsConfigured and NeedsReconnect are mutually exclusive, and exactly one
	// of them is true at any time.  The settings row branches on all three.

	table := []struct {
		status         string
		needsSetup     bool
		isConfigured   bool
		needsReconnect bool
	}{
		{"", true, false, false},
		{UserConnectionStatusReady, false, true, false},
		{UserConnectionStatusReconnect, false, false, true},
	}

	for _, test := range table {

		t.Run(test.status, func(t *testing.T) {

			userConnection := NewUserConnection()
			userConnection.Status = test.status

			require.Equal(t, test.needsSetup, userConnection.NeedsSetup())
			require.Equal(t, test.isConfigured, userConnection.IsConfigured())
			require.Equal(t, test.needsReconnect, userConnection.NeedsReconnect())
		})
	}
}

// TestUserConnection_HasWebhook confirms the flag that D40 leaves behind when an install fails
func TestUserConnection_HasWebhook(t *testing.T) {

	// A webhook that would not install leaves an EMPTY webhookId rather than blocking the
	// connection, and this is the only trace of that -- read by the settings row and by the
	// disconnect path that has to know whether there is anything to remove.

	userConnection := NewUserConnection()
	require.False(t, userConnection.HasWebhook())

	userConnection.Data.SetString(UserConnectionDataWebhookID, "wh1")
	require.True(t, userConnection.HasWebhook())
}

// TestUserConnection_HasNoFieldsProjection guards the trap that would make a configured
// connection read as unconfigured
func TestUserConnection_HasNoFieldsProjection(t *testing.T) {

	// QueryBuilder projects with T.Fields(), and a projection omitting `vault` would hand
	// working code a connection with no credential in it. Having no projection at all is
	// what makes that impossible; MerchantAccount's projection omits exactly that field.
	var object any = NewUserConnection()

	_, hasFields := object.(interface{ Fields() []string })

	require.False(t, hasFields, "UserConnection must not define a Fields() projection")
}
