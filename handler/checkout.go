package handler

import (
	"io"
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
)

// GetCheckout initiates a checkout session with the provided MerchantAccount and Product.
func GetCheckout(ctx *steranko.Context, factory *domain.Factory, merchantAccount *model.MerchantAccount, product *model.Product) error {

	const location = "handler.GetCheckout"

	// Create a "checkout" session, and generate a URL where the user will checkout
	returnURL := ctx.QueryParam("return")
	merchantAccountService := factory.MerchantAccount()
	checkoutURL, err := merchantAccountService.GetCheckoutURL(merchantAccount, product, returnURL)

	if err != nil {
		return derp.Wrap(err, location, "Error retrieving checkout URL")
	}

	// Forward the client to the checkout URL
	return ctx.Redirect(http.StatusTemporaryRedirect, checkoutURL)
}

// GetCheckoutResopnse collects the confirmation data from a successful checkout and updates Guest/Purchase records accordingly.
func GetCheckoutResponse(ctx *steranko.Context, factory *domain.Factory, merchantAccount *model.MerchantAccount, product *model.Product) error {

	const location = "handler.GetCheckoutResponse"

	// Verify the Checkout Session
	merchantAccountService := factory.MerchantAccount()
	privilege, err := merchantAccountService.ParseCheckoutResponse(merchantAccount, product, ctx.QueryParam("transactionId"), ctx.QueryParams())

	if err != nil {
		return derp.Wrap(err, location, "Error retrieving checkout URL")
	}

	authorization := getAuthorization(ctx)

	// If the guest is not already logged in, then lets log them in now
	if authorization.IdentityID != privilege.IdentityID {

		// Fall through means we need to update their JWT/Cookie
		authorization.IdentityID = privilege.IdentityID

		if err := factory.Steranko().SetCookie(ctx, authorization); err != nil {
			return derp.Wrap(err, location, "Error setting guest authorization")
		}
	}

	// Forward the client to their profile page and highlight the newly purchased privilege.
	return ctx.Redirect(http.StatusSeeOther, "/@guest")
}

// PostCheckoutWebhook processes inbound webhook events for a specific MerchantAccount
func PostCheckoutWebhook(ctx *steranko.Context, factory *domain.Factory, merchantAccount *model.MerchantAccount) error {

	const location = "handler.PostCheckoutWebhook"

	// Read the request Body as a byte array
	reader := io.LimitReader(ctx.Request().Body, 65535)
	body, err := io.ReadAll(reader)

	if err != nil {
		return derp.Wrap(err, location, "Error reading request body")
	}

	defer ctx.Request().Body.Close()

	// Parse the WebHook data based on the MerchantAccount type
	merchantAccountService := factory.MerchantAccount()

	if err := merchantAccountService.ParseCheckoutWebhook(ctx.Request().Header, body, merchantAccount); err != nil {

		// Suppress errors from unsupported event handlers
		if derp.IsNotImplemented(err) {
			return nil
		}

		// Suppress errors from subscriptions that are not found on this server
		if derp.IsNotFound(err) {
			return nil
		}

		// All other errors are reported to the caller
		return derp.Wrap(err, location, "Error processing webhook data")
	}

	// Success.  WebHook complete.
	return ctx.NoContent(http.StatusOK)
}
