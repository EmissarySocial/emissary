package service

import (
	"net/url"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/random"
	api "github.com/EmissarySocial/emissary/tools/stripeapi"
	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/remote/options"
	"github.com/benpate/rosetta/mapof"
	"github.com/davecgh/go-spew/spew"
	"github.com/golang-jwt/jwt/v5"
)

// stripe_getCheckoutURL generates a URL where users can purchase a product from Stripe.
func (service *MerchantAccount) stripe_getCheckoutURL(merchantAccount *model.MerchantAccount, remoteProductID string, returnURL string) (string, error) {

	const location = "service.MerchantAccount.stripe_getCheckoutURL"
	restrictedKey, err := service.stripe_getRestrictedKey(merchantAccount)

	if err != nil {
		return "", derp.Wrap(err, location, "Error retrieving restricted key")
	}

	// Load the Price/Prooduct from the Stripe API
	price, err := api.Price(restrictedKey, remoteProductID)

	if err != nil {
		return "", derp.Wrap(err, location, "Error retrieving price from Stripe")
	}

	// Send checkout session to the Stripe API
	checkoutResult := mapof.NewAny()
	transactionID, err := random.GenerateString(32)

	if err != nil {
		return "", derp.Wrap(err, location, "Error generating transaction ID")
	}

	// Wrap the parameters in a JWT token
	claims := jwt.MapClaims{
		"iat":               time.Now().Unix(),
		"merchantAccountId": merchantAccount.MerchantAccountID.Hex(),
		"productId":         remoteProductID,
		"transactionId":     transactionID,
	}

	// If the merchant account is in live mode, set the expiration to 1 hour (but not for dev/test)
	if merchantAccount.LiveMode {
		claims["exp"] = time.Now().Add(time.Hour * 1).Unix()
	}

	// Create and sign the JWT token
	token, err := service.jwtService.NewToken(claims)

	if err != nil {
		return "", derp.Wrap(err, location, "Error generating JWT token")
	}

	// Create a new Stripe Checkout Session
	successURL := service.host + "/.checkout/response?checkoutSessionId={CHECKOUT_SESSION_ID}&return=" + url.QueryEscape(returnURL) + "&jwt=" + token

	txn := remote.Post("https://api.stripe.com/v1/checkout/sessions").
		With(options.BearerAuth(restrictedKey)).
		ContentType("application/x-www-form-urlencoded").
		Form("mode", service.stripe_checkoutMode(price)).
		Form("line_items[0][price]", price.ID).
		Form("line_items[0][quantity]", "1").
		Form("ui_mode", "hosted").
		Form("client_reference_id", transactionID).
		Form("customer_creation", "always").
		Form("cancel_url", returnURL).
		Form("success_url", successURL).
		Result(&checkoutResult)

	if err := txn.Send(); err != nil {
		return "", derp.Wrap(err, location, "Error connecting to Stripe API")
	}

	// Return the URL to the caller
	return checkoutResult.GetString("url"), nil
}

// stripe_parseCheckoutResponse parses the response from a Stripe Checkout Session.
func (service *MerchantAccount) stripe_getGuestFromCheckoutResponse(queryParams url.Values, merchantAccount *model.MerchantAccount) (model.Guest, error) {

	const location = "service.MerchantAccount.stripe_parseCheckoutResponse"

	// Collect the CheckoutSessionID from the request.
	// This value was passed in a JWT, and unpacked by the WithMerchantAccount middleware, so it can be trusted
	checkoutSessionID := queryParams.Get("checkoutSessionId")

	// Get API Keys from the vault
	restrictedKey, err := service.stripe_getRestrictedKey(merchantAccount)

	if err != nil {
		return model.Guest{}, derp.Wrap(err, location, "Error retrieving API keys")
	}

	// Load the Checkout session from the Stripe API so that we can validate it and retrieve the customer details
	checkoutSession, err := api.CheckoutSession(restrictedKey, checkoutSessionID)
	if err != nil {
		return model.Guest{}, derp.Wrap(err, location, "Error loading checkout session from Stripe")
	}

	spew.Dump(checkoutSession)

	// RULE: transaction id must match the checkout session
	if checkoutSession.ClientReferenceID != queryParams.Get("transactionId") {
		return model.Guest{}, derp.BadRequestError(location, "Invalid Transaction ID", "The transaction ID does not match the checkout session")
	}

	// RULE: customer details must be present
	if checkoutSession.CustomerDetails == nil {
		return model.Guest{}, derp.BadRequestError(location, "Invalid Checkout Session", "The checkout session does not contain customer details")
	}

	// Use customer details to load or create a Guest record
	emailAddress := checkoutSession.CustomerDetails.Email
	remoteGuestID := checkoutSession.Customer.ID

	guest, err := service.guestService.LoadOrCreate(emailAddress, model.MerchantAccountTypeStripe, remoteGuestID)

	if err != nil {
		return model.Guest{}, derp.Wrap(err, location, "Error loading/creating guest by email", "email: "+emailAddress, "customerId: "+remoteGuestID)
	}

	// Success.
	return guest, nil
}
