package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
)

// GetCheckout initiates a checkout session with the provided MerchantAccount and Subscription.
func GetCheckout(ctx *steranko.Context, factory *domain.Factory, merchantAccount *model.MerchantAccount, subscription *model.Subscription) error {

	const location = "handler.GetCheckout"

	// Create a "checkout" session, and generate a URL where the user will checkout
	returnURL := ctx.Request().URL.Query().Get("return")
	merchantAccountService := factory.MerchantAccount()
	checkoutURL, err := merchantAccountService.GetCheckoutURL(merchantAccount, subscription, returnURL, returnURL)

	if err != nil {
		return derp.Wrap(err, location, "Error retrieving checkout URL")
	}

	// Forward the client to the checkout URL
	return ctx.Redirect(http.StatusTemporaryRedirect, checkoutURL)
}

// PostCheckoutWebhook receives webhook events from MerchantAccounts and processes them.
func PostCheckoutWebhook(ctx *steranko.Context, factory *domain.Factory, merchantAccount *model.MerchantAccount) error {

	const location = "handler.PostCheckoutWebhook"

	// Parse the WebHook data based on the MerchantAccount type
	merchantAccountService := factory.MerchantAccount()
	subscribers, err := merchantAccountService.ParseCheckoutWebhook(ctx.Request(), merchantAccount)

	if err != nil {

		// Suppress errors from unsupported events
		if derp.ErrorCode(err) == http.StatusBadRequest {
			if derp.Message(err) == "Unsupported Event" {
				return nil
			}
		}

		return derp.Wrap(err, location, "Error processing webhook data")
	}

	// Load/Create a subscriber record
	subscriberService := factory.Subscriber()

	for _, subscriber := range subscribers {

		if err := subscriberService.CreateOrUpdate(&subscriber); err != nil {
			return derp.Wrap(err, location, "Error loading or creating subscriber")
		}

	}

	return ctx.NoContent(http.StatusOK)
}
