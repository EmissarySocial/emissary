package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
)

/******************************************
 * MerchantAccount Export
 *
 * The export is handed to a third-party app that
 * holds a portability grant, so the property here
 * is privacy: the vault stays home. A Stripe Connect
 * vault holds the domain's platform key, which the
 * merchant never owned (BUG-303, Defect D).
 ******************************************/

// TestMerchantAccountExportDocument_OmitsVault is BUG-303 Defect D: the export used to decrypt
// the whole vault into the document, platform key included
func TestMerchantAccountExportDocument_OmitsVault(t *testing.T) {

	// The Connection the account was linked through, published on the cached Domain record
	connection := model.NewConnection()
	connection.ProviderID = model.ConnectionProviderStripeConnect
	connection.Type = model.ConnectionTypeUserPayment
	connection.Active = true

	domainService := NewDomain()
	writableDomain := model.NewWritableDomain()
	writableDomain.Connections[connection.ProviderID] = connection
	domainService.publish(writableDomain)

	connectionService := Connection{domainService: &domainService, masterKey: testDomainCipher}

	// A Stripe Connect account whose sealed vault holds the platform key
	merchantAccount := newTestMerchantAccount()
	merchantAccount.ConnectionID = connection.ConnectionID
	merchantAccount.Vault = newSealedVault(t, "restrictedKey", "rk_live_CANARY-PLATFORM-KEY")
	merchantAccount.Plaintext.SetString("accountId", "acct_CANARY")

	service := MerchantAccount{encryptionKey: testDomainCipher, connectionService: &connectionService}
	session := merchantAccountSession{merchantAccount: &merchantAccount}

	document, err := service.ExportDocument(session, merchantAccount.UserID, merchantAccount.MerchantAccountID)
	require.NoError(t, err)

	// The vault stays home, decrypted or sealed
	require.NotContains(t, document, "rk_live_CANARY-PLATFORM-KEY", "the export must not carry the decrypted vault")
	require.NotContains(t, document, `"Vault"`)

	for _, ciphertext := range merchantAccount.Vault.Encrypted {
		require.NotContains(t, document, ciphertext, "the export must not carry the sealed vault either")
	}

	// The export still carries the settings it exists to move
	require.Contains(t, document, "acct_CANARY", "Plaintext travels with the User")
	require.Contains(t, document, "Test Account")
	require.Contains(t, document, model.ConnectionProviderStripeConnect)
}

// merchantAccountSession is a data.Session whose every collection serves one MerchantAccount
type merchantAccountSession struct {
	brokenSession
	merchantAccount *model.MerchantAccount
}

// Collection returns a collection that loads the one MerchantAccount, whatever the criteria
func (session merchantAccountSession) Collection(string) data.Collection {
	return merchantAccountCollection{merchantAccount: session.merchantAccount}
}

// merchantAccountCollection serves one MerchantAccount to Load, and nothing else
type merchantAccountCollection struct {
	data.Collection
	merchantAccount *model.MerchantAccount
}

// Load copies the one record into target, whatever the criteria. The loader's own criteria
// are not under test here, so a fixed record keeps the fixture small.
func (collection merchantAccountCollection) Load(_ exp.Expression, target data.Object, _ ...option.Option) error {

	record, ok := target.(*model.MerchantAccount)

	if !ok {
		return derp.Internal("service.merchantAccountCollection.Load", "Target must be a MerchantAccount")
	}

	*record = *collection.merchantAccount
	return nil
}
