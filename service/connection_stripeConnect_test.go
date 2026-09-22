package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service/providers"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

/******************************************
 * Stripe Connect Repair
 *
 * StripeConnect.Connect writes the webhook secret that Stripe returns
 * into the vault, so Connection.Save must seal the vault after the
 * provider runs.  A fake provider stands in for Stripe here; the real
 * provider's Stripe calls are tested in service/providers.
 ******************************************/

// fakeProvider is a Provider whose Connect runs a function the test supplies
type fakeProvider struct {
	connect func(connection *model.Connection, vault mapof.String, host string) error
}

// BeforeSave does nothing
func (provider fakeProvider) BeforeSave(_ *model.Connection, _ mapof.String) error {
	return nil
}

// Connect runs the test's function, when there is one
func (provider fakeProvider) Connect(connection *model.Connection, vault mapof.String, host string) error {

	if provider.connect == nil {
		return nil
	}

	return provider.connect(connection, vault, host)
}

// Refresh does nothing
func (provider fakeProvider) Refresh(_ *model.Connection, _ mapof.String) error {
	return nil
}

// Disconnect does nothing
func (provider fakeProvider) Disconnect(_ *model.Connection, _ mapof.String) error {
	return nil
}

// addWebhookSecret is a Connect that does what StripeConnect's does: it writes a new webhook and its secret
func addWebhookSecret(connection *model.Connection, _ mapof.String, _ string) error {
	connection.Data.SetString("webhook", "we_new")
	connection.Vault.SetString("webhookSecret", "whsec_new")
	return nil
}

// useFakeStripeConnect makes the Connection service use a fake in place of the Stripe Connect provider
func useFakeStripeConnect(service *Connection, provider fakeProvider) {
	service.providerService.overrides = map[string]providers.Provider{
		model.ConnectionProviderStripeConnect: provider,
	}
}

// newStripeConnectConnection returns an active Stripe Connect Connection that uses vault
func newStripeConnectConnection(vault model.Vault) model.Connection {

	connection := model.NewConnection()
	connection.ProviderID = model.ConnectionProviderStripeConnect
	connection.Type = model.ConnectionTypeUserPayment
	connection.Active = true
	connection.Vault = vault

	return connection
}

// storeConnection writes a Connection onto the stored Domain record, and publishes it
func storeConnection(t *testing.T, domainService *Domain, session data.Session, connection model.Connection) {

	t.Helper()

	domain := model.NewWritableDomain()
	require.NoError(t, domainService.Load(session, &domain))
	domain.Connections[connection.ProviderID] = connection
	require.NoError(t, domainService.Save(session, &domain, "Stored a test Connection"))
}

// loadStoredConnection returns a Connection as the database holds it
func loadStoredConnection(t *testing.T, domainService *Domain, session data.Session, providerID string) model.Connection {

	t.Helper()

	stored := model.NewWritableDomain()
	require.NoError(t, domainService.Load(session, &stored))

	return stored.Connections[providerID]
}

// newRepairCheckService returns a Connection service whose cached Domain holds these Connections
func newRepairCheckService(host string, connections ...model.Connection) *Connection {

	domainService := NewDomain()
	domain := model.NewWritableDomain()

	for _, connection := range connections {
		domain.Connections[connection.ProviderID] = connection
	}

	domainService.publish(domain)

	return &Connection{domainService: &domainService, host: host}
}

// TestConnection_Save_SealsSecretsThatConnectAdds is the regression test for BUG-179: a secret the
// provider writes during Connect must reach the database sealed, beside the ones the form posted.
func TestConnection_Save_SealsSecretsThatConnectAdds(t *testing.T) {

	connectionService, domainService, session := newTestConnectionService(t)

	// The provider must still see the key that the form posted, before anything is sealed
	var seenKey string

	useFakeStripeConnect(connectionService, fakeProvider{connect: func(connection *model.Connection, vault mapof.String, host string) error {
		seenKey = vault.GetString("restrictedKey")
		return addWebhookSecret(connection, vault, host)
	}})

	connection := newStripeConnectConnection(model.NewVault())
	connection.Vault.SetString("restrictedKey", "rk_test_posted")

	require.NoError(t, connectionService.Save(session, &connection, "Connected"))

	stored := loadStoredConnection(t, domainService, session, model.ConnectionProviderStripeConnect)
	require.Equal(t, "rk_test_posted", seenKey, "Connect sees the key the form posted")
	require.Equal(t, "we_new", stored.Data.GetString("webhook"))
	require.Equal(t, "whsec_new", openStoredVault(t, stored.Vault, "webhookSecret"), "the secret Connect added is stored, sealed")
	require.Equal(t, "rk_test_posted", openStoredVault(t, stored.Vault, "restrictedKey"), "the key the form posted is stored, sealed")
}

// TestConnection_StripeConnectNeedsRepair pins which Stripe Connect Connections the startup check repairs
func TestConnection_StripeConnectNeedsRepair(t *testing.T) {

	withWebhook := func(vault model.Vault) model.Connection {
		connection := newStripeConnectConnection(vault)
		connection.Data.SetString("webhook", "we_old")
		return connection
	}

	t.Run("NoConnection", func(t *testing.T) {
		require.False(t, newRepairCheckService("https://example.com").StripeConnectNeedsRepair())
	})

	t.Run("Inactive", func(t *testing.T) {
		connection := withWebhook(model.NewVault())
		connection.Active = false
		require.False(t, newRepairCheckService("https://example.com", connection).StripeConnectNeedsRepair())
	})

	t.Run("LocalHostname", func(t *testing.T) {
		connection := withWebhook(model.NewVault())
		require.False(t, newRepairCheckService("http://localhost", connection).StripeConnectNeedsRepair())
	})

	t.Run("SecretIsStored", func(t *testing.T) {
		connection := withWebhook(newSealedVault(t, "webhookSecret", "whsec_old"))
		require.False(t, newRepairCheckService("https://example.com", connection).StripeConnectNeedsRepair())
	})

	t.Run("SecretIsMissing", func(t *testing.T) {
		connection := withWebhook(newSealedVault(t, "restrictedKey", "rk_test_1"))
		require.True(t, newRepairCheckService("https://example.com", connection).StripeConnectNeedsRepair())
	})

	t.Run("NoWebhookYet", func(t *testing.T) {
		connection := newStripeConnectConnection(newSealedVault(t, "restrictedKey", "rk_test_1"))
		require.True(t, newRepairCheckService("https://example.com", connection).StripeConnectNeedsRepair(), "a Connection made on a local hostname registers its webhook once it is public")
	})

	t.Run("OtherProvidersAreIgnored", func(t *testing.T) {
		connection := withWebhook(model.NewVault())
		connection.ProviderID = model.ConnectionProviderGiphy
		require.False(t, newRepairCheckService("https://example.com", connection).StripeConnectNeedsRepair())
	})
}

// TestConnection_RepairStripeConnect confirms a Connection with no stored secret is saved again, which
// stores the secret that Connect adds, and that a repaired Connection is left alone.
func TestConnection_RepairStripeConnect(t *testing.T) {

	t.Run("RepairsAMissingSecret", func(t *testing.T) {

		connectionService, domainService, session := newTestConnectionService(t)
		useFakeStripeConnect(connectionService, fakeProvider{connect: addWebhookSecret})

		broken := newStripeConnectConnection(newSealedVault(t, "restrictedKey", "rk_test_1"))
		broken.Data.SetString("webhook", "we_old")
		storeConnection(t, domainService, session, broken)
		require.True(t, connectionService.StripeConnectNeedsRepair())

		repaired, err := connectionService.RepairStripeConnect(session)
		require.NoError(t, err)
		require.True(t, repaired)

		stored := loadStoredConnection(t, domainService, session, model.ConnectionProviderStripeConnect)
		require.Equal(t, "we_new", stored.Data.GetString("webhook"))
		require.Equal(t, "whsec_new", openStoredVault(t, stored.Vault, "webhookSecret"))
		require.Equal(t, "rk_test_1", openStoredVault(t, stored.Vault, "restrictedKey"), "the existing key survives the repair")

		// Repaired once, and never again
		require.False(t, connectionService.StripeConnectNeedsRepair())

		repaired, err = connectionService.RepairStripeConnect(session)
		require.NoError(t, err)
		require.False(t, repaired)
	})

	t.Run("NothingToRepair", func(t *testing.T) {

		connectionService, _, session := newTestConnectionService(t)

		calls := 0
		useFakeStripeConnect(connectionService, fakeProvider{connect: func(*model.Connection, mapof.String, string) error {
			calls++
			return nil
		}})

		repaired, err := connectionService.RepairStripeConnect(session)
		require.NoError(t, err)
		require.False(t, repaired)
		require.Zero(t, calls, "nothing is sent to Stripe")
	})

	t.Run("ConnectFails", func(t *testing.T) {

		connectionService, domainService, session := newTestConnectionService(t)
		useFakeStripeConnect(connectionService, fakeProvider{connect: func(*model.Connection, mapof.String, string) error {
			return derp.Internal("test", "Stripe refused the webhook")
		}})

		broken := newStripeConnectConnection(newSealedVault(t, "restrictedKey", "rk_test_1"))
		broken.Data.SetString("webhook", "we_old")
		storeConnection(t, domainService, session, broken)

		repaired, err := connectionService.RepairStripeConnect(session)
		require.Error(t, err)
		require.False(t, repaired)
		require.True(t, connectionService.StripeConnectNeedsRepair(), "a failed repair is tried again at the next boot")
	})
}
