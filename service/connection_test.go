package service

import (
	"context"
	"testing"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	mockdb "github.com/benpate/data-mock"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

/******************************************
 * Untested Paths
 *
 * Providers come from a fixed switch, but a test can substitute a fake
 * through Provider.overrides (see connection_stripeConnect_test.go, which
 * also covers a failing Connect).  No registered provider's BeforeSave or
 * Disconnect ever fails, so those error returns in Save and Delete are
 * untested.  Vault.Encrypt cannot fail once DecodeMasterKey has accepted
 * the key.
 ******************************************/

// newTestConnectionDomain returns a Domain record holding one Giphy Connection that uses vault
func newTestConnectionDomain(vault model.Vault) model.WritableDomain {

	connection := model.NewConnection()
	connection.ProviderID = model.ConnectionProviderGiphy
	connection.Type = model.ConnectionTypeImage
	connection.Active = true
	connection.Data["clientId"] = "original"
	connection.Data["liveMode"] = "SANDBOX"
	connection.Vault = vault

	writableDomain := model.NewWritableDomain()
	writableDomain.Hostname = "example.com"
	writableDomain.Connections[connection.ProviderID] = connection
	return writableDomain
}

// newPlaceholderVault returns a Vault holding values that are never decrypted
func newPlaceholderVault() model.Vault {
	return model.Vault{
		Encrypted: mapof.String{"apiKey": "original-ciphertext"},
		Nonces:    mapof.String{"apiKey": "original-nonce"},
	}
}

// newSealedVault returns a Vault sealed with testDomainCipher, shaped as a database read returns it
func newSealedVault(t *testing.T, name string, value string) model.Vault {

	t.Helper()

	key, err := config.DecodeMasterKey(testDomainCipher)
	require.NoError(t, err)

	vault := model.NewVault()
	vault.SetString(name, value)
	require.NoError(t, vault.Encrypt(key))

	// A decoded record carries no plaintext, so keep only the stored fields
	return model.Vault{Encrypted: vault.Encrypted, Nonces: vault.Nonces}
}

// openStoredVault decrypts one value from a Vault's stored fields, ignoring any unsaved plaintext
func openStoredVault(t *testing.T, vault model.Vault, name string) string {

	t.Helper()

	key, err := config.DecodeMasterKey(testDomainCipher)
	require.NoError(t, err)

	stored := model.Vault{Encrypted: vault.Encrypted, Nonces: vault.Nonces, Nonce: vault.Nonce}
	values, err := stored.Decrypt(key, name)
	require.NoError(t, err)

	return values[name]
}

// newTestConnectionService returns a Connection service whose Domain record, holding one sealed
// Giphy Connection, is stored in an in-memory database, plus a session that reaches it.
func newTestConnectionService(t *testing.T) (*Connection, *Domain, data.Session) {

	t.Helper()

	session, err := mockdb.New().Session(context.Background())
	require.NoError(t, err)

	domainService := NewDomain()
	writableDomain := newTestConnectionDomain(newSealedVault(t, "apiKey", "original-secret"))
	require.NoError(t, domainService.Save(session, &writableDomain, "Created"))

	connectionService := Connection{
		domainService:   &domainService,
		providerService: &Provider{},
		masterKey:       testDomainCipher,
		host:            "https://example.com",
	}

	return &connectionService, &domainService, session
}

// TestConnection_ReadsCurrentDomain pins that the Connection service reads whatever Domain record
// is published NOW, rather than the one that was cached when it was wired up.
func TestConnection_ReadsCurrentDomain(t *testing.T) {

	domainService := NewDomain()
	connectionService := Connection{domainService: &domainService}

	// Nothing is connected yet
	count, err := connectionService.Count(nil, exp.All())
	require.Nil(t, err)
	require.Zero(t, count)

	// Another server connects Stripe, and the watcher publishes its record
	writableDomain := model.NewWritableDomain()
	writableDomain.Connections["STRIPE"] = model.Connection{ProviderID: "STRIPE", Type: "PAYMENT", Active: true}
	domainService.publish(writableDomain)

	count, err = connectionService.Count(nil, exp.All())
	require.Nil(t, err)
	require.Equal(t, int64(1), count)

	var result model.Connection
	require.Nil(t, connectionService.Load(nil, exp.Equal("providerId", "STRIPE"), &result))
	require.Equal(t, "STRIPE", result.ProviderID)

	require.Len(t, connectionService.ActiveByType("PAYMENT"), 1)
	require.Len(t, connectionService.AllAsMap(nil), 1)
}

// TestConnection_Load_ReturnsACopy pins that editing a loaded Connection never reaches the
// published Domain record.  The edit-connection step, NewOAuthClient, and Vault.Encrypt all write.
func TestConnection_Load_ReturnsACopy(t *testing.T) {

	domainService := NewDomain()
	connectionService := Connection{domainService: &domainService}
	domainService.publish(newTestConnectionDomain(newPlaceholderVault()))

	var loaded model.Connection
	require.NoError(t, connectionService.LoadByProvider(nil, model.ConnectionProviderGiphy, &loaded))

	// Write into every map a caller can reach through the loaded value
	loaded.Data["clientId"] = "edited"
	loaded.Vault.Encrypted["apiKey"] = "edited-ciphertext"
	loaded.Vault.Nonces["apiKey"] = "edited-nonce"
	delete(loaded.Data, "liveMode")

	cached := domainService.Cached().Connections[model.ConnectionProviderGiphy]
	require.Equal(t, "original", cached.Data["clientId"])
	require.Equal(t, "SANDBOX", cached.Data["liveMode"])
	require.Equal(t, "original-ciphertext", cached.Vault.Encrypted["apiKey"])
	require.Equal(t, "original-nonce", cached.Vault.Nonces["apiKey"])

	// The copy still carries every value it was loaded with
	require.Equal(t, "edited", loaded.Data["clientId"])
	require.Equal(t, model.ConnectionTypeImage, loaded.Type)
}

// TestConnection_Load_NilMapsStayNil pins that cloning keeps a missing map missing
func TestConnection_Load_NilMapsStayNil(t *testing.T) {

	domainService := NewDomain()
	connectionService := Connection{domainService: &domainService}

	writableDomain := model.NewWritableDomain()
	writableDomain.Connections[model.ConnectionProviderGiphy] = model.Connection{ProviderID: model.ConnectionProviderGiphy}
	domainService.publish(writableDomain)

	var loaded model.Connection
	require.NoError(t, connectionService.LoadByProvider(nil, model.ConnectionProviderGiphy, &loaded))
	require.Nil(t, loaded.Data)
	require.Nil(t, loaded.Vault.Encrypted)
	require.Nil(t, loaded.Vault.Nonces)
}

// TestConnection_Load_NotFound pins the error kind for a provider with no Connection
func TestConnection_Load_NotFound(t *testing.T) {

	domainService := NewDomain()
	connectionService := Connection{domainService: &domainService}

	var loaded model.Connection
	err := connectionService.LoadByProvider(nil, model.ConnectionProviderGiphy, &loaded)
	require.True(t, derp.IsNotFound(err), "got %v", err)
}

// storeInvalidDomain rewrites the stored Domain record so that its next validation fails, standing
// in for a database error on the write that follows a Load
func storeInvalidDomain(t *testing.T, session data.Session) {

	t.Helper()

	invalid := loadStoredDomain(t, session)
	invalid.ColorMode = "NOT-A-COLOR-MODE"
	require.NoError(t, session.Collection("Domain").Save(&invalid, "Invalid"))
}

// TestConnection_WritesStartFromTheStoredRecord pins that Save and Delete edit the record in the
// database, not the cached one, so a stale cache is never written back.
func TestConnection_WritesStartFromTheStoredRecord(t *testing.T) {

	t.Run("Save", func(t *testing.T) {

		connectionService, domainService, session := newTestConnectionService(t)

		// The cache falls behind the database, as it does between a save elsewhere and the watcher
		stale := model.NewWritableDomain()
		stale.Label = "Stale"
		domainService.publish(stale)

		connection, err := connectionService.LoadOrCreateByProvider(session, model.ConnectionProviderUnsplash)
		require.NoError(t, err)

		connection.Type = model.ConnectionTypeImage
		require.NoError(t, connectionService.Save(session, &connection, "test"))

		// The stored record keeps everything the stale cache lacked, and the cache catches up
		stored := loadStoredDomain(t, session)
		require.Equal(t, "example.com", stored.Hostname)
		require.Len(t, stored.Connections, 2)
		require.Equal(t, "example.com", domainService.Cached().Hostname)
		require.Len(t, domainService.Cached().Connections, 2)
	})

	t.Run("Delete", func(t *testing.T) {

		connectionService, domainService, session := newTestConnectionService(t)

		stale := model.NewWritableDomain()
		stale.Label = "Stale"
		domainService.publish(stale)

		connection := model.NewConnection()
		connection.ProviderID = model.ConnectionProviderGiphy
		require.NoError(t, connectionService.Delete(session, &connection, "test"))

		stored := loadStoredDomain(t, session)
		require.Equal(t, "example.com", stored.Hostname)
		require.Empty(t, stored.Connections)
		require.Equal(t, "example.com", domainService.Cached().Hostname)
	})

	t.Run("NilConnections", func(t *testing.T) {

		connectionService, domainService, session := newTestConnectionService(t)

		// A record stored before Connections existed decodes with a nil map
		stored := loadStoredDomain(t, session)
		stored.Connections = nil
		require.NoError(t, session.Collection("Domain").Save(&stored, "No connections"))

		connection, err := connectionService.LoadOrCreateByProvider(session, model.ConnectionProviderUnsplash)
		require.NoError(t, err)

		connection.Type = model.ConnectionTypeImage
		require.NoError(t, connectionService.Save(session, &connection, "test"))
		require.Len(t, domainService.Cached().Connections, 1)
	})

	t.Run("LoadFails", func(t *testing.T) {

		connectionService, domainService, _ := newTestConnectionService(t)
		before := domainService.Cached()

		connection := model.NewConnection()
		connection.ProviderID = model.ConnectionProviderGiphy
		connection.Type = model.ConnectionTypeImage

		require.Error(t, connectionService.Save(failingSession{}, &connection, "test"))
		require.Error(t, connectionService.Delete(failingSession{}, &connection, "test"))
		require.Same(t, before, domainService.Cached())
	})
}

// TestConnection_Save pins that a saved Connection is sealed, written into the stored Domain
// record, and published, and that every refusal leaves the cached record as it was.
func TestConnection_Save(t *testing.T) {

	t.Run("UpdatesAnExistingConnection", func(t *testing.T) {

		connectionService, domainService, session := newTestConnectionService(t)

		connection, err := connectionService.LoadOrCreateByProvider(session, model.ConnectionProviderGiphy)
		require.NoError(t, err)

		connection.Data["clientId"] = "updated"
		connection.Vault.SetString("apiKey", "updated-secret")
		require.NoError(t, connectionService.Save(session, &connection, "test"))

		// The stored record holds the new values, with the secret sealed
		stored := loadStoredDomain(t, session).Connections[model.ConnectionProviderGiphy]
		require.Equal(t, "updated", stored.Data["clientId"])
		require.Equal(t, "updated-secret", openStoredVault(t, stored.Vault, "apiKey"))

		// ...and so does the published record
		cached := domainService.Cached().Connections[model.ConnectionProviderGiphy]
		require.Equal(t, "updated", cached.Data["clientId"])
		require.Equal(t, "updated-secret", openStoredVault(t, cached.Vault, "apiKey"))
	})

	t.Run("AddsANewConnection", func(t *testing.T) {

		connectionService, domainService, session := newTestConnectionService(t)

		connection, err := connectionService.LoadOrCreateByProvider(session, model.ConnectionProviderUnsplash)
		require.NoError(t, err)

		connection.Type = model.ConnectionTypeImage
		connection.Active = true
		require.NoError(t, connectionService.Save(session, &connection, "test"))

		stored := loadStoredDomain(t, session)
		require.Len(t, stored.Connections, 2)
		require.Contains(t, stored.Connections, model.ConnectionProviderUnsplash)
		require.Contains(t, stored.Connections, model.ConnectionProviderGiphy)
		require.Len(t, domainService.Cached().Connections, 2)
	})

	t.Run("InactiveConnectionIsStored", func(t *testing.T) {

		connectionService, domainService, session := newTestConnectionService(t)

		connection, err := connectionService.LoadOrCreateByProvider(session, model.ConnectionProviderGiphy)
		require.NoError(t, err)

		connection.Active = false
		require.NoError(t, connectionService.Save(session, &connection, "test"))

		require.False(t, loadStoredDomain(t, session).Connections[model.ConnectionProviderGiphy].Active)
		require.False(t, domainService.Cached().Connections[model.ConnectionProviderGiphy].Active)
	})

	t.Run("RejectedEditLeavesCacheUnchanged", func(t *testing.T) {

		connectionService, domainService, session := newTestConnectionService(t)

		connection, err := connectionService.LoadOrCreateByProvider(session, model.ConnectionProviderGiphy)
		require.NoError(t, err)

		// Edit the loaded copy the way the edit-connection step does, then fail validation
		// AFTER the vault has been sealed
		connection.Data["clientId"] = "rejected"
		connection.Vault.SetString("apiKey", "rejected-secret")
		connection.Type = "NOT-A-TYPE"
		require.Error(t, connectionService.Save(session, &connection, "test"))

		cached := domainService.Cached().Connections[model.ConnectionProviderGiphy]
		require.Equal(t, "original", cached.Data["clientId"])
		require.Equal(t, "original-secret", openStoredVault(t, cached.Vault, "apiKey"))
		require.Equal(t, "original", loadStoredDomain(t, session).Connections[model.ConnectionProviderGiphy].Data["clientId"])
	})

	t.Run("UndecryptableVault", func(t *testing.T) {

		connectionService, domainService, session := newTestConnectionService(t)
		before := domainService.Cached()

		connection, err := connectionService.LoadOrCreateByProvider(session, model.ConnectionProviderGiphy)
		require.NoError(t, err)

		connection.Vault.Encrypted["apiKey"] = "not-hexadecimal"
		require.Error(t, connectionService.Save(session, &connection, "test"))
		require.Same(t, before, domainService.Cached())
	})

	t.Run("DomainWriteFails", func(t *testing.T) {

		connectionService, domainService, session := newTestConnectionService(t)

		storeInvalidDomain(t, session)
		before := domainService.Cached()

		connection, err := connectionService.LoadOrCreateByProvider(session, model.ConnectionProviderGiphy)
		require.NoError(t, err)

		connection.Data["clientId"] = "unsaved"
		require.Error(t, connectionService.Save(session, &connection, "test"))
		require.Same(t, before, domainService.Cached())
		require.Equal(t, "original", domainService.Cached().Connections[model.ConnectionProviderGiphy].Data["clientId"])
	})

	t.Run("UnknownProvider", func(t *testing.T) {

		connectionService, domainService, session := newTestConnectionService(t)
		before := domainService.Cached()

		connection := model.NewConnection()
		connection.ProviderID = "NOT-A-PROVIDER"
		require.Error(t, connectionService.Save(session, &connection, "test"))
		require.Same(t, before, domainService.Cached())
	})

	t.Run("MissingMasterKey", func(t *testing.T) {

		connectionService, domainService, session := newTestConnectionService(t)
		connectionService.masterKey = ""
		before := domainService.Cached()

		connection := model.NewConnection()
		connection.ProviderID = model.ConnectionProviderGiphy
		require.Error(t, connectionService.Save(session, &connection, "test"))
		require.Same(t, before, domainService.Cached())
	})
}

// TestConnection_Delete pins that a deleted Connection leaves both the stored and the published
// Domain record, and that a refusal changes neither.
func TestConnection_Delete(t *testing.T) {

	t.Run("RemovesTheConnection", func(t *testing.T) {

		connectionService, domainService, session := newTestConnectionService(t)

		connection, err := connectionService.LoadOrCreateByProvider(session, model.ConnectionProviderGiphy)
		require.NoError(t, err)

		require.NoError(t, connectionService.Delete(session, &connection, "test"))
		require.Empty(t, loadStoredDomain(t, session).Connections)
		require.Empty(t, domainService.Cached().Connections)
	})

	t.Run("UnknownProvider", func(t *testing.T) {

		connectionService, domainService, session := newTestConnectionService(t)

		connection := model.NewConnection()
		connection.ProviderID = "NOT-A-PROVIDER"
		require.Error(t, connectionService.Delete(session, &connection, "test"))
		require.Len(t, domainService.Cached().Connections, 1)
	})

	t.Run("MissingMasterKey", func(t *testing.T) {

		connectionService, domainService, session := newTestConnectionService(t)
		connectionService.masterKey = ""

		connection, err := connectionService.LoadOrCreateByProvider(session, model.ConnectionProviderGiphy)
		require.NoError(t, err)

		require.Error(t, connectionService.Delete(session, &connection, "test"))
		require.Len(t, loadStoredDomain(t, session).Connections, 1)
		require.Len(t, domainService.Cached().Connections, 1)
	})

	t.Run("DomainWriteFails", func(t *testing.T) {

		connectionService, domainService, session := newTestConnectionService(t)

		storeInvalidDomain(t, session)
		before := domainService.Cached()

		connection, err := connectionService.LoadOrCreateByProvider(session, model.ConnectionProviderGiphy)
		require.NoError(t, err)

		require.Error(t, connectionService.Delete(session, &connection, "test"))
		require.Same(t, before, domainService.Cached())

		// The stored record is not checked here: data-mock hands Load the stored map itself
		// (BUG-178), so TestDomain_Save/RejectedSaveLeavesTheDatabaseUnchanged covers it
	})
}
