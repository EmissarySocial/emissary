package handler

import (
	"io"
	"net/http"
	"strings"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
)

// GetCheckout initiates a checkout session with the provided MerchantAccount and Product.
func GetCheckout(ctx *steranko.Context, factory *domain.Factory) error {

	const location = "handler.GetCheckout"

	// Get Parameters from the QueryString
	token := ctx.QueryParam("token")
	tokenArgs := strings.SplitN(token, ":", 3)

	if len(tokenArgs) != 3 {
		return derp.InternalError(
			location,
			"Invalid token format. Expected format: tokenType:merchantAccountID:remoteProductID",
			token,
		)
	}

	tokenType := tokenArgs[0]
	merchantAccountID := tokenArgs[1]
	remoteProductID := tokenArgs[2]

	if tokenType != "MA" {
		return derp.InternalError(
			location,
			"Invalid token type. Expected 'MA' for Merchant Account",
			token,
		)
	}

	// Load the MerchantAccount
	merchantAccount := model.NewMerchantAccount()
	if err := factory.MerchantAccount().LoadByToken(merchantAccountID, &merchantAccount); err != nil {
		return derp.Wrap(err, location, "Error loading MerchantAccount")
	}

	// Create a "checkout" session, and generate a URL where the user will checkout
	returnURL := ctx.QueryParam("return")
	merchantAccountService := factory.MerchantAccount()
	checkoutURL, err := merchantAccountService.GetCheckoutURL(&merchantAccount, remoteProductID, returnURL)

	if err != nil {
		return derp.Wrap(err, location, "Error retrieving checkout URL")
	}

	// Forward the client to the checkout URL
	return ctx.Redirect(http.StatusTemporaryRedirect, checkoutURL)
}

// GetCheckoutResopnse collects the confirmation data from a successful checkout and updates Guest/Purchase records accordingly.
func GetCheckoutResponse(ctx *steranko.Context, factory *domain.Factory, merchantAccount *model.MerchantAccount) error {

	const location = "handler.GetCheckoutResponse"

	// Verify the Checkout Session
	merchantAccountService := factory.MerchantAccount()
	privilege, err := merchantAccountService.ParseCheckoutResponse(ctx.QueryParams(), merchantAccount)

	if err != nil {
		return derp.Wrap(err, location, "Error retrieving checkout URL")
	}

	authorization := getAuthorization(ctx)

	// If the guest is not already logged in, then lets log them in now
	if authorization.IdentityID != privilege.IdentityID {

		// Fall through means we need to update their JWT/Cookie
		authorization.IdentityID = privilege.IdentityID
		token, err := factory.JWT().NewToken(authorization)

		if err != nil {
			return derp.Wrap(err, location, "Error creating authorization token")
		}

		cookieName := steranko.CookieName(ctx.Request())
		isTLS := ctx.Request().TLS != nil
		cookie := factory.Steranko().CreateCookie(cookieName, token, isTLS)

		ctx.SetCookie(&cookie)
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
