package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
)

// PostCheckout initiates a checkout session with the provided MerchantAccount and Subscription.
func PostCheckout(ctx *steranko.Context, factory *domain.Factory, subscription *model.Subscription, merchantAccount *model.MerchantAccount) error {

	const location = "handler.PostCheckout"

	// Create a "checkout" session, and generate a URL where the user will checkout
	referer := ctx.Request().Header.Get("Referer")
	merchantAccountService := factory.MerchantAccount()
	checkoutURL, err := merchantAccountService.GetCheckoutURL(merchantAccount, subscription, referer, referer)

	if err != nil {
		return derp.Wrap(err, location, "Error retrieving checkout URL")
	}

	// Forward the client to the checkout URL
	ctx.Response().Header().Set("Hx-Redirect", checkoutURL)
	return ctx.NoContent(http.StatusOK)
}

// PostCheckoutWebhook receives webhook events from MerchantAccounts and processes them.
func PostCheckoutWebhook(ctx *steranko.Context, factory *domain.Factory, subscription *model.Subscription, merchantAccount *model.MerchantAccount) error {
	return nil
}
