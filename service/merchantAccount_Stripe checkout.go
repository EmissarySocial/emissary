package service

import (
	"net/url"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/random"
	"github.com/EmissarySocial/emissary/tools/stripeapi"
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
	checkoutMode := service.stripe_checkoutMode(price)

	txn := remote.Post("https://api.stripe.com/v1/checkout/sessions").
		With(options.BearerAuth(restrictedKey), options.Debug()).
		ContentType("application/x-www-form-urlencoded").
		Form("mode", checkoutMode).
		Form("line_items[0][price]", price.ID).
		Form("line_items[0][quantity]", "1").
		Form("ui_mode", "hosted").
		Form("client_reference_id", transactionID).
		Form("cancel_url", returnURL).
		Form("success_url", successURL).
		Result(&checkoutResult)

	// If this is a single payment (not a subscription), then we need to create a customer
	if checkoutMode == "payment" {
		txn.Form("customer_creation", "always")
	}

	// Send the transaction to Stripe
	if err := txn.Send(); err != nil {
		return "", derp.Wrap(err, location, "Error connecting to Stripe API")
	}

	spew.Dump(location, checkoutResult)

	// Return the URL to the caller
	return checkoutResult.GetString("url"), nil
}

// stripe_parseCheckoutResponse parses the response from a Stripe Checkout Session.
func (service *MerchantAccount) stripe_getPrivilegeFromCheckoutResponse(queryParams url.Values, merchantAccount *model.MerchantAccount) (model.Privilege, error) {

	const location = "service.MerchantAccount.stripe_parseCheckoutResponse"

	// Collect the CheckoutSessionID from the request.
	// This value was passed in a JWT, and unpacked by the WithMerchantAccount middleware, so it can be trusted
	checkoutSessionID := queryParams.Get("checkoutSessionId")

	// Get API Keys from the vault
	restrictedKey, err := service.stripe_getRestrictedKey(merchantAccount)

	if err != nil {
		return model.Privilege{}, derp.Wrap(err, location, "Error retrieving API keys")
	}

	// Load the Checkout session from the Stripe API so that we can validate it and retrieve the customer details
	checkoutSession, err := api.CheckoutSession(restrictedKey, checkoutSessionID)
	if err != nil {
		return model.Privilege{}, derp.Wrap(err, location, "Error loading checkout session from Stripe")
	}

	spew.Dump(location, checkoutSession)

	// RULE: transaction id must match the checkout session
	if checkoutSession.ClientReferenceID != queryParams.Get("transactionId") {
		return model.Privilege{}, derp.BadRequestError(location, "Invalid Transaction ID", "The transaction ID does not match the checkout session")
	}

	// RULE: customer details must be present
	if checkoutSession.CustomerDetails == nil {
		return model.Privilege{}, derp.BadRequestError(location, "Invalid Checkout Session", "The checkout session does not contain customer details")
	}

	// This is safe becuase it was passed via a JWT token (see WithMerchantAccountJWT)
	productID := queryParams.Get("productId")

	// Retrieve the Price/Product from the Stripe API
	price, err := stripeapi.Price(restrictedKey, productID)

	if err != nil {
		return model.Privilege{}, derp.Wrap(err, location, "Error retrieving price from Stripe", productID)
	}

	// Create a new Identity record for the guest
	identity, err := service.identityService.LoadOrCreate(
		checkoutSession.CustomerDetails.Name,
		model.IdentifierTypeEmail,
		checkoutSession.CustomerDetails.Email,
		true,
	)

	if err != nil {
		return model.Privilege{}, derp.Wrap(err, location, "Error saving Identity", identity)
	}

	// Create a privilege for this Identity
	remoteProductID := productID
	remotePersonID := checkoutSession.Customer.ID
	remotePurchaseID := checkoutSession.ID

	// Populate a new Privilege record for the Identity
	privilege := model.NewPrivilege()
	privilege.IdentityID = identity.IdentityID
	privilege.Name = price.Product.Name
	privilege.PriceDescription = service.stripe_priceLabel(price)
	privilege.RecurringType = service.stripe_recurringType(price)
	privilege.UserID = merchantAccount.UserID
	privilege.MerchantAccountID = merchantAccount.MerchantAccountID
	privilege.RemotePersonID = remotePersonID
	privilege.RemoteProductID = remoteProductID
	privilege.RemotePurchaseID = remotePurchaseID
	privilege.IdentifierType = model.IdentifierTypeEmail
	privilege.IdentifierValue = identity.EmailAddress

	if err := service.privilegeService.Save(&privilege, "Created via Stripe Checkout"); err != nil {
		return model.Privilege{}, derp.Wrap(err, location, "Error saving Privilege", privilege)
	}

	// Success.
	return privilege, nil
}
