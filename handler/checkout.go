package handler

import (
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
