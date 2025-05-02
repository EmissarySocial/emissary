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

// GetCheckout initiates a checkout session with the provided MerchantAccount and Subscription.
func GetCheckout(ctx *steranko.Context, factory *domain.Factory, merchantAccount *model.MerchantAccount, subscription *model.Subscription) error {

	const location = "handler.GetCheckout"

	// Create a "checkout" session, and generate a URL where the user will checkout
	returnURL := ctx.QueryParam("return")
	merchantAccountService := factory.MerchantAccount()
	checkoutURL, err := merchantAccountService.GetCheckoutURL(merchantAccount, subscription, returnURL)

	if err != nil {
		return derp.Wrap(err, location, "Error retrieving checkout URL")
	}

	// Forward the client to the checkout URL
	return ctx.Redirect(http.StatusTemporaryRedirect, checkoutURL)
}

// GetCheckoutResopnse initiates a checkout session with the provided MerchantAccount and Subscription.
func GetCheckoutResponse(ctx *steranko.Context, factory *domain.Factory, merchantAccount *model.MerchantAccount, subscription *model.Subscription) error {

	const location = "handler.GetCheckout"

	// Verify the Checkout Session
	merchantAccountService := factory.MerchantAccount()
	subscribers, err := merchantAccountService.ParseCheckoutResponse(ctx.QueryParams(), merchantAccount)

	if err != nil {
		return derp.Wrap(err, location, "Error retrieving checkout URL")
	}

	if subscribers.IsEmpty() {
		return derp.NewBadRequestError(location, "No subscribers found")
	}

	subscriber := subscribers.First()
	authorization := getAuthorization(ctx)
	spew.Dump(subscribers)
	spew.Dump(authorization)

	// If the subscriber is already logged in, then just forward them to the return URL
	if authorization.VisitorEmail == subscriber.EmailAddress {
		return ctx.Redirect(http.StatusSeeOther, ctx.QueryParam("return"))
	}

	// Fall through means we need to update their JWT/Cookie
	authorization.VisitorEmail = subscriber.EmailAddress

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
	subscribers, err := merchantAccountService.ParseCheckoutWebhook(ctx.Request().Header, body, merchantAccount)

	if err != nil {

		// Suppress errors from unsupported events
		if derp.ErrorCode(err) == http.StatusBadRequest {
			if derp.Message(err) == "Unsupported Event" {
				return nil
			}
		}

		return derp.Wrap(err, location, "Error processing webhook data")
	}

	// Load/Create a subscriber record(s)
	subscriberService := factory.Subscriber()

	for _, subscriber := range subscribers {

		if err := subscriberService.CreateOrUpdate(&subscriber); err != nil {
			return derp.Wrap(err, location, "Error loading or creating subscriber")
		}

	}

	// Success.  WebHook complete.
	return ctx.NoContent(http.StatusOK)
}
