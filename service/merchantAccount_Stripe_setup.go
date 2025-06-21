package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/domain"
	"github.com/benpate/remote"
	"github.com/benpate/remote/options"
	"github.com/benpate/rosetta/mapof"
)

// stripe_refreshMerchantAccount ensures that the Stripe webhook is configured for this MerchantAccount
func (service *MerchantAccount) stripe_refreshMerchantAccount(merchantAccount *model.MerchantAccount) error {

	const location = "service.MerchantAccount.stripe_refreshMerchantAccount"

	// Cannot set webhooks for local domains
	if domain.IsLocalhost(service.host) {
		return nil
	}

	// Guarantee that a webhook has been configured for this MerchantAccount
	if merchantAccount.Plaintext.GetString("webhook") == "" {

		// Get API Keys from the vault
		restrictedKey, err := service.stripe_getRestrictedKey(merchantAccount)

		if err != nil {
			return derp.Wrap(err, location, "Error retrieving API keys")
		}

		endpoint := service.host + "/.checkout/webhook?merchantAccountId=" + merchantAccount.MerchantAccountID.Hex()

		// Configure a new Webhook in the Stripe API
		webhookResult := mapof.NewAny()
		txn := remote.Post("https://api.stripe.com/v1/webhook_endpoints").
			With(options.BearerAuth(restrictedKey)).
			Query("url", endpoint).
			Query("description", domain.NameOnly(service.host)+" supscription updates").
			Query("enabled_events[]", "checkout.session.completed").
			Query("enabled_events[]", "customer.product.created").
			Query("enabled_events[]", "customer.product.deleted").
			Query("enabled_events[]", "customer.product.paused").
			Query("enabled_events[]", "customer.product.resumed").
			Query("enabled_events[]", "customer.product.updated").
			Result(&webhookResult)

		if connectedAccountID := merchantAccount.Plaintext.GetString("accountId"); connectedAccountID != "" {
			txn.Header("Stripe-Account", connectedAccountID)
		}
		if err := txn.Send(); err != nil {
			return derp.Wrap(err, location, "Error connecting to Stripe API")
		}

		// Save the webhook data into the MerchantAccount
		merchantAccount.Plaintext.SetString("webhook", webhookResult.GetString("id"))
		merchantAccount.Vault.SetString("webhookSecret", webhookResult.GetString("secret"))
	}

	return nil
}
