package service

import (
	"net/url"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/random"
	"github.com/EmissarySocial/emissary/tools/stripeapi"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/remote/options"
	"github.com/benpate/rosetta/mapof"
	"github.com/golang-jwt/jwt/v5"
)

// stripe_getCheckoutURL generates a URL where users can purchase a product from Stripe.
func (service *MerchantAccount) stripe_getCheckoutURL(merchantAccount *model.MerchantAccount, product *model.Product, returnURL string) (string, error) {

	const location = "service.MerchantAccount.stripe_getCheckoutURL"
	restrictedKey, err := service.stripe_getRestrictedKey(merchantAccount)

	if err != nil {
		return "", derp.Wrap(err, location, "Error retrieving restricted key")
	}

	connectedAccountID := service.stripe_getConnectedAccountID(merchantAccount)

	// Load the Price/Prooduct from the Stripe API
	price, err := stripeapi.Price(restrictedKey, connectedAccountID, product.RemoteID)

	if err != nil {
		return "", derp.Wrap(err, location, "Error retrieving price from Stripe")
	}

	// Send checkout session to the Stripe API
	checkoutResult := mapof.NewAny()
	transactionID, err := random.GenerateString(32)

	if err != nil {
		return "", derp.Wrap(err, location, "Unable to generate transaction ID")
	}

	// Wrap the parameters in a JWT token
	claims := jwt.MapClaims{
		"iat":           time.Now().Unix(),
		"productId":     product.ProductID.Hex(),
		"transactionId": transactionID,
	}

	// If the merchant account is in live mode, set the expiration to 1 hour (but not for dev/test)
	if merchantAccount.LiveMode {
		claims["exp"] = time.Now().Add(time.Hour * 1).Unix()
	}

	// Create and sign the JWT token
	token, err := service.jwtService.NewToken(claims)

	if err != nil {
		return "", derp.Wrap(err, location, "Unable to generate JWT token")
	}

	// Create a new Stripe Checkout Session
	successURL := service.host + "/.checkout/response?checkoutSessionId={CHECKOUT_SESSION_ID}&return=" + url.QueryEscape(returnURL) + "&jwt=" + token
	checkoutMode := service.stripe_checkoutMode(price)

	txn := remote.Post("https://api.stripe.com/v1/checkout/sessions").
		With(options.BearerAuth(restrictedKey)).
		With(stripeapi.ConnectedAccount(connectedAccountID)).
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

	// Return the URL to the caller
	return checkoutResult.GetString("url"), nil
}

// stripe_parseCheckoutResponse parses the response from a Stripe Checkout Session.
func (service *MerchantAccount) stripe_getPrivilegeFromCheckoutResponse(session data.Session, merchantAccount *model.MerchantAccount, product *model.Product, transactionID string, queryParams url.Values) (model.Privilege, error) {

	const location = "service.MerchantAccount.stripe_parseCheckoutResponse"

	// Collect the CheckoutSessionID from the request.
	// This value was passed in a JWT, and unpacked by the WithMerchantAccount middleware, so it can be trusted
	checkoutSessionID := queryParams.Get("checkoutSessionId")

	// Get API Keys from the vault
	restrictedKey, err := service.stripe_getRestrictedKey(merchantAccount)

	if err != nil {
		return model.Privilege{}, derp.Wrap(err, location, "Error retrieving API keys")
	}

	connectedAccountID := service.stripe_getConnectedAccountID(merchantAccount)

	// Load the Checkout session from the Stripe API so that we can validate it and retrieve the customer details
	checkoutSession, err := stripeapi.CheckoutSession(restrictedKey, connectedAccountID, checkoutSessionID)
	if err != nil {
		return model.Privilege{}, derp.Wrap(err, location, "Unable to load checkout session from Stripe")
	}

	// RULE: transaction id must match the checkout session
	if checkoutSession.ClientReferenceID != transactionID {
		return model.Privilege{}, derp.BadRequest(location, "Invalid Transaction ID", "The transaction ID does not match the checkout session")
	}

	// RULE: customer details must be present
	if checkoutSession.CustomerDetails == nil {
		return model.Privilege{}, derp.BadRequest(location, "Invalid Checkout Session", "The checkout session does not contain customer details")
	}

	// Retrieve the Price/Product from the Stripe API
	price, err := stripeapi.Price(restrictedKey, connectedAccountID, product.RemoteID)

	if err != nil {
		return model.Privilege{}, derp.Wrap(err, location, "Error retrieving price from Stripe", product.RemoteID)
	}

	// Create a new Identity record for the guest
	identity, err := service.identityService.LoadOrCreate(
		session,
		checkoutSession.CustomerDetails.Name,
		model.IdentifierTypeEmail,
		checkoutSession.CustomerDetails.Email,
	)

	if err != nil {
		return model.Privilege{}, derp.Wrap(err, location, "Unable to save Identity", identity)
	}

	// Create a privilege for this Identity
	remotePurchaseID := checkoutSession.ID

	if checkoutSession.Subscription != nil {
		remotePurchaseID = checkoutSession.Subscription.ID
	}

	// Populate a new Privilege record for the Identity
	privilege := model.NewPrivilege()
	privilege.IdentityID = identity.IdentityID
	privilege.ProductID = product.ProductID
	privilege.Name = price.Product.Name
	privilege.PriceDescription = service.stripe_priceLabel(price)
	privilege.RecurringType = service.stripe_recurringType(price)
	privilege.UserID = merchantAccount.UserID
	privilege.MerchantAccountID = merchantAccount.MerchantAccountID
	privilege.RemotePersonID = checkoutSession.Customer.ID
	privilege.RemoteProductID = product.RemoteID
	privilege.RemotePurchaseID = remotePurchaseID
	privilege.IdentifierType = model.IdentifierTypeEmail
	privilege.IdentifierValue = identity.EmailAddress
	privilege.IsVisible = true

	if err := service.privilegeService.Save(session, &privilege, "Created via Stripe Checkout"); err != nil {
		return model.Privilege{}, derp.Wrap(err, location, "Unable to save Privilege", privilege)
	}

	// Success.
	return privilege, nil
}

func (service *MerchantAccount) stripe_CancelPrivilege(merchantAccount *model.MerchantAccount, privilege *model.Privilege) error {

	const location = "service.MerchantAccount.stripe_CancelPrivilege"

	// Get API Keys from the vault
	restrictedKey, err := service.stripe_getRestrictedKey(merchantAccount)

	if err != nil {
		return derp.Wrap(err, location, "Error retrieving API keys")
	}

	connectedAccountID := service.stripe_getConnectedAccountID(merchantAccount)

	// Call the Stripe API to cancel the subscription
	if err := stripeapi.SubscriptionCancel(restrictedKey, connectedAccountID, privilege.RemotePurchaseID); err != nil {
		return derp.Wrap(err, location, "Error canceling subscription")
	}

	// Success.
	return nil
}
