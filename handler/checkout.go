package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
)

// GetCheckout initiates a checkout session with the provided MerchantAccount and Product.
func GetCheckout(ctx *steranko.Context, factory *service.Factory, session data.Session, merchantAccount *model.MerchantAccount, product *model.Product) error {

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

// GetCheckoutResponse collects the confirmation data from a successful checkout and updates Guest/Purchase records accordingly.
func GetCheckoutResponse(ctx *steranko.Context, factory *service.Factory, session data.Session, merchantAccount *model.MerchantAccount, product *model.Product) error {

	const location = "handler.GetCheckoutResponse"

	// Verify the Checkout Session
	merchantAccountService := factory.MerchantAccount()
	privilege, err := merchantAccountService.ParseCheckoutResponse(session, merchantAccount, product, ctx.QueryParam("transactionId"), ctx.QueryParams())

	if err != nil {
		return derp.Wrap(err, location, "Error retrieving checkout URL")
	}

	// If the guest is not already logged in, then lets log them in now
	if authorization := getAuthorization(ctx); authorization.IdentityID != privilege.IdentityID {

		// Fall through means we need to update their JWT/Cookie
		authorization.IdentityID = privilege.IdentityID

		if err := factory.Steranko(session).SetCookie(ctx, authorization); err != nil {
			return derp.Wrap(err, location, "Unable to set guest authorization")
		}
	}

	// Forward the client to their profile page and highlight the newly purchased privilege.
	return ctx.Redirect(http.StatusSeeOther, "/@guest")
}
