package handler

import (
	"io"
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"github.com/davecgh/go-spew/spew"
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

	const location = "handler.GetCheckout"

	// Verify the Checkout Session
	merchantAccountService := factory.MerchantAccount()
	guest, err := merchantAccountService.ParseCheckoutResponse(ctx.QueryParams(), merchantAccount)

	if err != nil {
		return derp.Wrap(err, location, "Error retrieving checkout URL")
	}

	authorization := getAuthorization(ctx)

	// If the purchase is already logged in, then just forward them to the return URL
	if authorization.GuestID == guest.GuestID {
		return ctx.Redirect(http.StatusSeeOther, ctx.QueryParam("return"))
	}

	// Fall through means we need to update their JWT/Cookie
	authorization.GuestID = guest.GuestID
	token, err := factory.JWT().NewToken(authorization)

	if err != nil {
		return derp.Wrap(err, location, "Error creating authorization token")
	}

	cookieName := steranko.CookieName(ctx.Request())
	isTLS := ctx.Request().TLS != nil
	cookie := factory.Steranko().CreateCookie(cookieName, token, isTLS)

	ctx.SetCookie(&cookie)
	spew.Dump(cookie)

	// Forward the client to the checkout URL
	returnURL := ctx.QueryParam("return")
	return ctx.Redirect(http.StatusSeeOther, returnURL)
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

		// Suppress errors from unsupported events
		if derp.IsNotImplemented(err) {
			return nil
		}

		// All other errors are reported to the caller
		return derp.Wrap(err, location, "Error processing webhook data")
	}

	// Success.  WebHook complete.
	return ctx.NoContent(http.StatusOK)
}
