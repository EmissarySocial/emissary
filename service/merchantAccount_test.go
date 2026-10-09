package service

import (
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/secretcheck"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestMerchantAccount_OmitsVaultFromErrors requires that no failure reports an account's vault
func TestMerchantAccount_OmitsVaultFromErrors(t *testing.T) {

	// BUG-173: the vault is json:"-" but bson:"vault", so these errors stored its ciphertext,
	// which the MasterKey leaked elsewhere would decrypt.
	service := MerchantAccount{encryptionKey: testDomainCipher}

	// requireSite fails unless err came from the named site, carrying none of the vault
	requireSite := func(t *testing.T, err error, merchantAccount model.MerchantAccount, location string, message string) {
		t.Helper()
		require.Equal(t, location, derp.Location(err))
		require.Equal(t, message, derp.Message(err))

		for _, secret := range merchantAccount.Vault.Encrypted {
			secretcheck.RequireAbsent(t, err, secret)
		}

		secretcheck.RequireAbsent(t, err, "merchant-n0nce")
	}

	t.Run("Save", func(t *testing.T) {
		merchantAccount := newTestMerchantAccount()
		err := service.Save(brokenSession{}, &merchantAccount, "test")
		requireSite(t, err, merchantAccount, "service.MerchantAccount.Save", "Saving MerchantAccount")
	})

	t.Run("Delete", func(t *testing.T) {
		merchantAccount := newTestMerchantAccount()
		err := service.Delete(brokenSession{}, &merchantAccount, "test")
		requireSite(t, err, merchantAccount, "service.MerchantAccount.Delete", "Deleting MerchantAccount")
	})

	t.Run("DeleteByUserID", func(t *testing.T) {
		merchantAccount := newTestMerchantAccount()
		session := brokenSession{records: map[string][]data.Object{"MerchantAccount": {&merchantAccount}}}
		err := service.DeleteByUserID(session, merchantAccount.UserID, "test")
		requireSite(t, err, merchantAccount, "service.MerchantAccount.DeleteByUserID", "Deleting MerchantAccount")
	})

	t.Run("RemoteProductsByUser", func(t *testing.T) {
		merchantAccount := newTestMerchantAccount()
		merchantAccount.Type = "NOT-A-PROCESSOR"
		session := brokenSession{records: map[string][]data.Object{"MerchantAccount": {&merchantAccount}}}
		_, _, err := service.RemoteProductsByUser(session, merchantAccount.UserID)
		requireSite(t, err, merchantAccount, "service.MerchantAccount.RemoteProductsByUser", "Loading products for merchant account")
	})
}

// newTestMerchantAccount returns an account whose vault holds ciphertext, and whose API key
// does not expire soon, so Connect never reaches the payment processor
func newTestMerchantAccount() model.MerchantAccount {
	merchantAccount := model.NewMerchantAccount()
	merchantAccount.UserID = primitive.NewObjectID()
	merchantAccount.Type = model.ConnectionProviderStripeConnect
	merchantAccount.Name = "Test Account"
	merchantAccount.APIKeyExpirationDate = time.Now().Add(24 * time.Hour).Unix()
	merchantAccount.Vault.Encrypted["apiKey"] = "merchant-ciph3rtext"
	merchantAccount.Vault.Nonces["apiKey"] = "merchant-n0nce"
	return merchantAccount
}
