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

// GetCheckoutResopnse initiates a checkout session with the provided MerchantAccount and Product.
func GetCheckoutResponse(ctx *steranko.Context, factory *domain.Factory, merchantAccount *model.MerchantAccount, product *model.Product) error {

	const location = "handler.GetCheckout"

	// Verify the Checkout Session
	merchantAccountService := factory.MerchantAccount()
	guest, purchases, err := merchantAccountService.ParseCheckoutResponse(ctx.QueryParams(), merchantAccount)

	if err != nil {
		return derp.Wrap(err, location, "Error retrieving checkout URL")
	}

	if purchases.IsEmpty() {
		return derp.BadRequestError(location, "No purchases found")
	}

	purchase := purchases.First()

	authorization := getAuthorization(ctx)
	spew.Dump(purchases)
	spew.Dump(authorization)

	// If the purchase is already logged in, then just forward them to the return URL
	if authorization.VisitorEmail == guest.EmailAddress {
		return ctx.Redirect(http.StatusSeeOther, ctx.QueryParam("return"))
	}

	// Fall through means we need to update their JWT/Cookie
	authorization.GuestEmail = guest.EmailAddress

	jwtService := factory.JWT()

	token, err := jwtService.NewToken(authorization)

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

// PostCheckoutWebhook receives webhook events from MerchantAccounts and processes them.
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
	purchases, err := merchantAccountService.ParseCheckoutWebhook(ctx.Request().Header, body, merchantAccount)

	if err != nil {

		// Suppress errors from unsupported events
		if derp.ErrorCode(err) == http.StatusBadRequest {
			if derp.Message(err) == "Unsupported Event" {
				return nil
			}
		}

		return derp.Wrap(err, location, "Error processing webhook data")
	}

	// Load/Create a purchase record(s)
	purchaseService := factory.Purchase()

	for _, purchase := range purchases {

		if err := purchaseService.CreateOrUpdate(&purchase); err != nil {
			return derp.Wrap(err, location, "Error loading or creating purchase")
		}

	}

	// Success.  WebHook complete.
	return ctx.NoContent(http.StatusOK)
}
