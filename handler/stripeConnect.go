package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/remote/options"
	"github.com/benpate/steranko"
	"github.com/stripe/stripe-go/v78"
)

// GetStripeConnect initiates the Stripe connection process for a User.
func GetStripeConnect(ctx *steranko.Context, factory *domain.Factory, user *model.User) error {

	const location = "handler.GetStripeConnect"

	// Load the Connection from the database
	connectionService := factory.Connection()
	connection := model.NewConnection()

	if err := connectionService.LoadByProvider(model.ConnectionProviderStripeConnect, &connection); err != nil {
		return derp.Wrap(err, location, "Error loading Stripe-Connect Connection")
	}

	vault, err := connectionService.DecryptVault(&connection)

	if err != nil {
		return derp.Wrap(err, location, "Error decrypting Stripe-Connect Connection Vault")
	}

	// Create a MerchantAccount for this User
	merchantAccountService := factory.MerchantAccount()
	merchantAccount := model.NewMerchantAccount()
	merchantAccount.UserID = user.UserID
	merchantAccount.Type = model.ConnectionProviderStripeConnect
	merchantAccount.LiveMode = connection.Data.GetString("liveMode") == "LIVE"
	merchantAccount.Name = "Stripe Connect"
	merchantAccount.Vault.SetString("restrictedKey", vault.GetString("restrictedKey"))

	// Create a new ACCOUNT on Stripe
	stripeAccount := stripe.Account{}

	accountTransaction := remote.Post("https://api.stripe.com/v1/accounts").
		Form("controller[fees][payer]", "account").           // Merchant is responsible for fees
		Form("controller[losses][payments]", "stripe").       // Stripe is responsible for losses from this Merchant
		Form("controller[requirement_collection]", "stripe"). // Stripe is responsible for collecting requirements

		With(options.BasicAuth(vault.GetString("restrictedKey"), "")).
		Result(&stripeAccount)

	if err := accountTransaction.Send(); err != nil {
		return derp.Wrap(err, location, "Error sending referral to Stripe", derp.WithCode(http.StatusInternalServerError))
	}

	// Save the new MerchantAccount (including Stripe Account ID)
	merchantAccount.Plaintext.SetString("accountId", stripeAccount.ID)

	if err := merchantAccountService.Save(&merchantAccount, "Linked by User"); err != nil {
		return derp.Wrap(err, location, "Error creating MerchantAccount", derp.WithCode(http.StatusInternalServerError))
	}

	// Create a new ACCOUNT LINK on Stripe
	accountLink := stripe.AccountLink{}
	returnURL := factory.Host() + "/@me/settings/payments"
	refreshURL := factory.Host() + "/.stripe-connect/connect?merchantAccountId=" + merchantAccount.MerchantAccountID.Hex()

	accountLinkTransaction := remote.Post("https://api.stripe.com/v1/account_links").
		Form("account", stripeAccount.ID).
		Form("refresh_url", refreshURL).
		Form("return_url", returnURL).
		Form("type", "account_onboarding").
		With(options.BasicAuth(vault.GetString("restrictedKey"), "")).
		Result(&accountLink)

	if err := accountLinkTransaction.Send(); err != nil {
		return derp.Wrap(err, location, "Error sending referral to Stripe", derp.WithCode(http.StatusInternalServerError))
	}

	return ctx.Redirect(http.StatusFound, accountLink.URL)
}

// PostStripeConnectWebhook_Checkout processes inbound webhook events for a specific MerchantAccount
func PostStripeConnectWebhook_Checkout(ctx *steranko.Context, factory *domain.Factory, connection *model.Connection) error {

	const location = "handler.PostStripeConnectWebhook_Checkout"

	// RULE: Connection must be a STRIPE CONNECT account
	if connection.Type != model.ConnectionProviderStripeConnect {
		return derp.NotImplementedError(location, "Connection is not a Stripe Connect account")
	}

	// Retrieve the webhook signing key from the Vault
	vault, err := factory.Connection().DecryptVault(connection, "webhookSecret")

	if err != nil {
		return derp.Wrap(err, location, "Error decrypting webhook secret")
	}

	liveMode := connection.Data.GetString("liveMode") == "LIVE"

	// Process the Stripe WebHook data
	if err = stripe_ProcessWebhook(factory, ctx.Request(), vault.GetString("webhookSecret"), liveMode); err != nil {

		// Suppress errors from subscriptions that are not found on this server
		if derp.IsNotFound(err) {
			return nil
		}

		// All other errors are reported to the caller
		return derp.Wrap(err, location, "Error processing webhook data")
	}

	// Success. WebHook complete.
	return ctx.NoContent(http.StatusOK)
}
