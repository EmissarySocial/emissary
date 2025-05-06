package service

import (
	"encoding/hex"
	"net/http"
	"net/url"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/remote"
	"github.com/benpate/remote/options"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
	"github.com/davecgh/go-spew/spew"
)

func (service *MerchantAccount) paypal_getServerAddress(merchantAccount *model.MerchantAccount) string {

	if merchantAccount.LiveMode {
		return "https://api-m.paypal.com"
	} else {
		return "https://api-m.sandbox.paypal.com"
	}
}

// paypal_parseCheckoutWebhook processes product webhook events from Stripe
func (service *MerchantAccount) paypal_parseCheckoutWebhook(header http.Header, body []byte, merchantAccount *model.MerchantAccount) (model.Guest, []model.Purchase, error) {

	const location = "service.MerchantAccount.paypal_parseCheckoutWebhook"

	spew.Dump(location, merchantAccount, header, string(body))

	return model.Guest{}, nil, derp.NotImplementedError(location, "Not Implemented")
}

// paypal_refreshMerchantAccount connects/refreshes the PayPal merchant account data
func (service *MerchantAccount) paypal_refreshMerchantAccount(merchantAccount *model.MerchantAccount) error {

	const location = "service.MerchantAccount.paypal_refreshMerchantAccount"

	// Decode the encryption key (this should never fail)
	encryptionKey, err := hex.DecodeString(service.encryptionKey)

	if err != nil {
		return derp.Wrap(err, location, "Error decoding encryption key")
	}

	// Open the Vault to get the clientID and secret key
	vault, err := merchantAccount.Vault.Decrypt(encryptionKey)

	if err != nil {
		return derp.Wrap(err, location, "Error decrypting vault data")
	}

	// Collect variables
	clientID := vault.GetString("clientId")
	secretKey := vault.GetString("secretKey")
	result := make(mapof.Any)

	endpoint := service.paypal_getServerAddress(merchantAccount) + "/v1/oauth2/token"

	// Connect to the PayPal API
	txn := remote.Post(endpoint).
		With(options.BasicAuth(clientID, secretKey)).
		ContentType("application/x-www-form-urlencoded").
		Form("grant_type", "client_credentials").
		Result(&result)

	if err := txn.Send(); err != nil {
		return derp.Wrap(err, location, "Error connecting to PayPal to refresh API key")
	}

	// Update values in the vault
	merchantAccount.Vault.SetString("appId", result.GetString("app_id"))
	merchantAccount.Vault.SetString("accessToken", result.GetString("access_token"))
	merchantAccount.APIKeyExpirationDate = time.Now().Unix() + result.GetInt64("expires_in")

	// Re-encrypt the vault data
	if err := merchantAccount.Vault.Encrypt(encryptionKey); err != nil {
		return derp.Wrap(err, location, "Error encrypting vault data")
	}

	// Success!
	return nil
}

// paypal_refreshProduct refreshes the product data for a PayPal product
func (service *MerchantAccount) paypal_refreshProduct(merchantAccount *model.MerchantAccount, product *model.Product) error {

	spew.Dump(merchantAccount, product)
	return nil
}

func (service *MerchantAccount) paypal_getProducts(merchantAccount *model.MerchantAccount) ([]form.LookupCode, error) {

	const location = "service.MerchantAccount.paypal_getProducts"

	endpoint := service.paypal_getServerAddress(merchantAccount) + "/v1/billing/plans"
	txnResult := mapof.NewAny()

	// Get API Keys from the vault
	apiKeys, err := service.DecryptVault(merchantAccount, "accessToken")

	if err != nil {
		return nil, derp.Wrap(err, location, "Error retrieving API keys")
	}

	txn := remote.Get(endpoint).
		Query("sort_by", "create_time").
		Query("sort_order", "desc").
		ContentType("application/json").
		Header("Prefer", "return=representation").
		With(options.BearerAuth(apiKeys.GetString("accessToken"))).
		Result(&txnResult)

	if err := txn.Send(); err != nil {
		return nil, derp.Wrap(err, location, "Error connecting to PayPal API")
	}

	plans := txnResult.GetSliceOfAny("plans")
	result := make([]form.LookupCode, len(plans))

	for index, planAny := range plans {
		plan := mapof.Any(convert.MapOfAny(planAny))

		result[index] = form.LookupCode{
			Value:       plan.GetString("id"),
			Label:       plan.GetString("name"),
			Description: plan.GetString("description"),
		}
	}

	return result, nil
}

func (service *MerchantAccount) paypal_getCheckoutURL(merchantAccount *model.MerchantAccount, product *model.Product, returnURL string) (string, error) {

	const location = "service.MerchantAccount.paypal_getCheckoutURL"

	// Get API Keys from the vault
	apiKeys, err := service.DecryptVault(merchantAccount, "accessToken")

	if err != nil {
		return "", derp.Wrap(err, location, "Error retrieving API keys")
	}

	// Create the checkout URL
	endpoint := service.paypal_getServerAddress(merchantAccount) + "/v1/billing/products/" + product.RemoteID
	txnResult := mapof.NewAny()

	txn := remote.Post(endpoint).
		ContentType("application/json").
		With(options.BearerAuth(apiKeys.GetString("accessToken"))).
		Result(&txnResult)

	if err := txn.Send(); err != nil {
		return "", derp.Wrap(err, location, "Error connecting to PayPal API")
	}

	return txnResult.GetString("checkout_url"), nil
}

func (service *MerchantAccount) paypal_parseCheckoutResponse(queryParams url.Values, merchantAccount *model.MerchantAccount) (model.Guest, []model.Purchase, error) {
	return model.NewGuest(), nil, derp.NotImplementedError("service.MerchantAccount.paypal_parseCheckoutResponse", "Not Implemented")
}
