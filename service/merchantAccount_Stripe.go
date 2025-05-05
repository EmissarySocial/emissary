package service

import (
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/currency"
	"github.com/EmissarySocial/emissary/tools/random"
	"github.com/benpate/derp"
	"github.com/benpate/domain"
	"github.com/benpate/form"
	"github.com/benpate/remote"
	"github.com/benpate/remote/options"
	"github.com/benpate/rosetta/compare"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/webhook"
)

// stripe_refreshMerchantAccount ensures that the Stripe webhook is configured for this MerchantAccount
func (service *MerchantAccount) stripe_refreshMerchantAccount(merchantAccount *model.MerchantAccount) error {

	const location = "service.MerchantAccount.stripe_refreshMerchantAccount"

	// Cannot set webhooks for local domains
	if domain.IsLocalhost(service.host) {
		return nil
	}

	// Guarantee that a webhook has been configured for this MerchantAccount
	if merchantAccount.Plaintext.GetString("webhook") == "" {

		// Get API Keys from the vault
		restrictedKey, err := service.stripe_getRestrictedKey(merchantAccount)

		if err != nil {
			return derp.Wrap(err, location, "Error retrieving API keys")
		}

		endpoint := service.host + "/.checkout/webhook?merchantAccountId=" + merchantAccount.MerchantAccountID.Hex()

		// Configure a new Webhook in the Stripe API
		webhookResult := mapof.NewAny()
		txn := remote.Post("https://api.stripe.com/v1/webhook_endpoints").
			With(options.BearerAuth(restrictedKey)).
			Query("url", endpoint).
			Query("description", domain.NameOnly(service.host)+" supscription updates").
			Query("enabled_events[]", "checkout.session.completed").
			Query("enabled_events[]", "customer.product.created").
			Query("enabled_events[]", "customer.product.deleted").
			Query("enabled_events[]", "customer.product.paused").
			Query("enabled_events[]", "customer.product.resumed").
			Query("enabled_events[]", "customer.product.updated").
			Result(&webhookResult)

		if err := txn.Send(); err != nil {
			return derp.Wrap(err, location, "Error connecting to Stripe API")
		}

		// Save the webhook data into the MerchantAccount
		merchantAccount.Plaintext.SetString("webhook", webhookResult.GetString("id"))
		merchantAccount.Vault.SetString("webhookSecret", webhookResult.GetString("secret"))
	}

	return nil
}

func (service *MerchantAccount) stripe_getCheckoutURL(merchantAccount *model.MerchantAccount, product *model.Product, returnURL string) (string, error) {

	const location = "service.MerchantAccount.stripe_getCheckoutURL"
	restrictedKey, err := service.stripe_getRestrictedKey(merchantAccount)

	if err != nil {
		return "", derp.Wrap(err, location, "Error retrieving restricted key")
	}

	// Send checkout session to the Stripe API
	checkoutResult := mapof.NewAny()
	transactionID, err := random.GenerateString(32)

	if err != nil {
		return "", derp.Wrap(err, location, "Error generating transaction ID")
	}

	// Wrap the parameters in a JWT token
	claims := jwt.MapClaims{
		"iat":           time.Now().Unix(),
		"exp":           time.Now().Add(time.Hour * 1).Unix(),
		"userId":        product.UserID.Hex(),
		"productId":     product.ProductID.Hex(),
		"transactionId": transactionID,
	}

	token, err := service.jwtService.NewToken(claims)

	if err != nil {
		return "", derp.Wrap(err, location, "Error generating JWT token")
	}

	successURL := service.host + "/.checkout/response?checkoutSessionId={CHECKOUT_SESSION_ID}&return=" + url.QueryEscape(returnURL) + "&jwt=" + token

	txn := remote.Post("https://api.stripe.com/v1/checkout/sessions").
		With(options.BearerAuth(restrictedKey)).
		ContentType("application/x-www-form-urlencoded").
		Form("mode", iif((product.RecurringType == model.ProductRecurringTypeOnetime), "payment", "product")).
		Form("line_items[0][price]", product.RemoteID).
		Form("line_items[0][quantity]", "1").
		Form("ui_mode", "hosted").
		Form("client_reference_id", transactionID).
		Form("cancel_url", returnURL).
		Form("success_url", successURL).
		Result(&checkoutResult)

	if err := txn.Send(); err != nil {
		return "", derp.Wrap(err, location, "Error connecting to Stripe API")
	}

	// Return the URL to the caller
	return checkoutResult.GetString("url"), nil
}

func (service *MerchantAccount) stripe_parseCheckoutResponse(queryParams url.Values, merchantAccount *model.MerchantAccount) (model.Guest, []model.Purchase, error) {

	const location = "service.MerchantAccount.stripe_parseCheckoutResponse"

	// Collect the CheckoutSessionID from the request.
	// These values were passed in a JWT, and unpacked by the WithMerchantAccount middleware, so they can be trusted
	checkoutSessionID := queryParams.Get("checkoutSessionId")
	transactionID := queryParams.Get("transactionId")

	// Load the Checkout session from the Stripe API
	stripeCheckoutSession, err := service.stripe_getCheckoutSession(merchantAccount, checkoutSessionID)
	if err != nil {
		return model.Guest{}, nil, derp.Wrap(err, location, "Error loading checkout session from Stripe")
	}

	// Verify that the TransactionID matches the value from the Checkout Session.
	if transactionID != stripeCheckoutSession.ClientReferenceID {
		return model.Guest{}, nil, derp.NewBadRequestError(location, "Invalid Transaction ID", "The transaction ID does not match the checkout session")
	}

	// Map Stripe.ProductID(s) into Purchases
	return service.stripe_mapSubscriptions(merchantAccount, stripeCheckoutSession.Subscription)
}

// stripe_handleWebhook processes product webhook events from Stripe
func (service *MerchantAccount) stripe_parseCheckoutWebhook(header http.Header, body []byte, merchantAccount *model.MerchantAccount) (model.Guest, []model.Purchase, error) {

	const location = "service.MerchantAccount.stripe_handleWebhook"

	var stripeEvent stripe.Event
	var stripeSubscription stripe.Subscription

	// Parse the request body into a Stripe event
	switch merchantAccount.LiveMode {

	// In TEST mode, just unmarshal the body directly
	case false:

		if err := json.Unmarshal(body, &stripeEvent); err != nil {
			return model.Guest{}, nil, derp.Wrap(err, location, "Error unmarshalling webhook stripeEvent")
		}

	// In LIVE mode, use the Stripe library to validate event
	default:

		// Retrieve the webhook signing key from the Vault
		vault, err := service.DecryptVault(merchantAccount, "webhookSecret")

		if err != nil {
			return model.Guest{}, nil, derp.Wrap(err, location, "Error decrypting webhook secret")
		}

		// Parse and validate the Webhook event
		stripeEvent, err = webhook.ConstructEvent(body, header.Get("Stripe-Signature"), vault.GetString("webhookSecret"))

		if err != nil {
			return model.Guest{}, nil, derp.Wrap(err, location, "Error parsing webhook event")
		}
	}

	// Unpack the Product from the Webhook event
	if err := json.Unmarshal(stripeEvent.Data.Raw, &stripeSubscription); err != nil {
		return model.Guest{}, nil, derp.Wrap(err, location, "Error unmarshalling product data")
	}

	// Map products from the Webhook into Purchases
	return service.stripe_mapSubscriptions(merchantAccount, &stripeSubscription)
}

func (service *MerchantAccount) stripe_mapSubscriptions(merchantAccount *model.MerchantAccount, stripeSubscription *stripe.Subscription) (model.Guest, []model.Purchase, error) {

	const location = "service.MerchantAccount.stripe_mapSubscriptions"

	// NPE check: product.Customer
	if stripeSubscription.Customer == nil {
		return model.Guest{}, nil, derp.NewBadRequestError(location, "Invalid Customer", "The customer value must not be null")
	}

	// NPE check: product.Items
	if stripeSubscription.Items == nil {
		return model.Guest{}, nil, derp.NewBadRequestError(location, "Invalid Subscription", "Stripe Subscription cannot be null")
	}

	if len(stripeSubscription.Items.Data) == 0 {
		return model.Guest{}, nil, derp.NewBadRequestError(location, "Invalid Subscription", "Sripe Subscription must have at least one item")
	}

	// Load Stripe Customer record from the remote API
	customer, err := service.stripe_getCustomer(merchantAccount, stripeSubscription.Customer.ID)

	if err != nil {
		return model.Guest{}, nil, derp.Wrap(err, location, "Error loading customer from Stripe")
	}

	// Create/Update Purchase records for each "price" line item in the product
	purchases := make(sliceof.Object[model.Purchase], 0, len(stripeSubscription.Items.Data))

	for _, item := range stripeSubscription.Items.Data {

		// NPE Check: item.Price
		if item.Price == nil {
			return model.Guest{}, nil, derp.NewBadRequestError(location, "Invalid Product", "No price found in product item")
		}

		// Try to find the Price in the database
		product := model.NewProduct()
		if err := service.productService.LoadByRemoteID(item.Price.ID, &product); err != nil {
			return model.Guest{}, nil, derp.Wrap(err, location, "Error loading product by token", item.Price.ID)
		}

		// Create the new Purchase record
		purchase := model.NewPurchase()
		purchase.ProductID = product.ProductID
		purchase.UserID = product.UserID
		purchase.EmailAddress = customer.Email
		purchase.RemoteUserID = customer.ID
		purchase.AuthorizationCode = ""
		purchase.RemoteProductID = item.Price.ID
		purchase.RemotePurchaseID = stripeProduct.ID
		purchase.StartDate = stripeProduct.StartDate
		purchase.EndDate = stripeProduct.CurrentPeriodEnd
		purchase.RecurringType = model.ProductRecurringTypeOnetime

		switch stripeProduct.Status {

		case "active", "trialing", "incomplete", "past_due", "unpaid":
			purchase.StateID = model.PurchaseStateActive
		case "paused":
			purchase.StateID = model.PurchaseStatePaused
		case "canceled", "incomplete_expired":
			purchase.StateID = model.PurchaseStateCanceled
		default:
			purchase.StateID = model.PurchaseStateCanceled
		}

		if item.Price.Recurring != nil {

			switch item.Price.Recurring.Interval {

			case stripe.PriceRecurringIntervalDay:
				purchase.RecurringType = model.ProductRecurringTypeDaily

			case stripe.PriceRecurringIntervalWeek:
				purchase.RecurringType = model.ProductRecurringTypeWeekly

			case stripe.PriceRecurringIntervalMonth:
				purchase.RecurringType = model.ProductRecurringTypeMonthly

			case stripe.PriceRecurringIntervalYear:
				purchase.RecurringType = model.ProductRecurringTypeYearly
			}
		}

		// Append the Purchase to the purchases set
		purchases = append(purchases, purchase)
	}

	// Create/Load the Guest record for this purchase

	// Great success.
	return guest, purchases, nil
}

// stripe_refreshProduct refreshes the product data for a Stripe product
func (service *MerchantAccount) stripe_refreshProduct(merchantAccount *model.MerchantAccount, product *model.Product) error {

	const location = "service.MerchantAccount.stripe_refreshProduct"

	// Get API Keys from the vault
	restrictedKey, err := service.stripe_getRestrictedKey(merchantAccount)

	if err != nil {
		return derp.Wrap(err, location, "Error retrieving API keys")
	}

	// Load the associated Stripe `Price` record
	price := mapof.NewAny()
	txn := remote.Get("https://api.stripe.com/v1/prices/" + product.RemoteID).
		With(options.BearerAuth(restrictedKey)).
		Result(&price)

	if err := txn.Send(); err != nil {
		return derp.Wrap(err, location, "Error connecting to Stripe API")
	}

	// Set/Update the RecurringType for the product
	switch price.GetMap("recurring").GetString("interval") {

	case "day":
		product.RecurringType = model.ProductRecurringTypeDaily

	case "week":
		product.RecurringType = model.ProductRecurringTypeWeekly

	case "month":
		product.RecurringType = model.ProductRecurringTypeMonthly

	case "year":
		product.RecurringType = model.ProductRecurringTypeYearly

	default:
		product.RecurringType = model.ProductRecurringTypeOnetime
	}

	// Set/Update the Price label for the product
	product.Price = service.stripe_priceLabel(price)

	// Subbess.
	return nil
}

// stripe_getPrices retrieves all prices from the Stripe API and returns them as a list of LookupCodes
func (service *MerchantAccount) stripe_getPrices(merchantAccount *model.MerchantAccount) ([]form.LookupCode, error) {

	const location = "service.MerchantAccount.paypal_getProducts"

	txnResult := mapof.NewAny()

	// Get API Keys from the vault
	restrictedKey, err := service.stripe_getRestrictedKey(merchantAccount)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error retrieving API keys")
	}

	// Query the Stripe API for all Prices
	txn := remote.Get("https://api.stripe.com/v1/prices?expand[]=data.product").
		With(options.BearerAuth(restrictedKey)).
		Result(&txnResult)

	if err := txn.Send(); err != nil {
		return nil, derp.Wrap(err, location, "Error connecting to PayPal API")
	}

	// Map prices into LookupCodes, grouping by Product name
	prices := txnResult.GetSliceOfAny("data")

	result := make([]form.LookupCode, len(prices))
	for index, record := range prices {
		price := mapof.Any(convert.MapOfAny(record))

		result[index] = form.LookupCode{
			Group: price.GetMap("product").GetString("name"),
			Value: price.GetString("id"),
			Label: service.stripe_priceLabel(price),
		}
	}

	// Sort the results by Group then Label
	slices.SortFunc(result, func(a, b form.LookupCode) int {

		if firstSort := compare.String(a.Group, b.Group); firstSort != 0 {
			return firstSort
		}

		return compare.String(a.Label, b.Label)
	})

	// Phew! Done.
	return result, nil
}

// stripe_getRestrictedKey retrieves the restricted API key for the specified MerchantAccount
func (service *MerchantAccount) stripe_getRestrictedKey(merchantAccount *model.MerchantAccount) (string, error) {

	const location = "service.MerchantAccount.stripe_getRestrictedKey"

	var propertyName string

	if merchantAccount.LiveMode {
		propertyName = "restrictedKey_live"
	} else {
		propertyName = "restrictedKey_test"
	}

	apiKeys, err := service.DecryptVault(merchantAccount, propertyName)

	if err != nil {
		return "", derp.Wrap(err, location, "Error retrieving API keys")
	}

	return apiKeys.GetString(propertyName), nil
}

// https://docs.stripe.com/api/checkout/sessions/object
func (service *MerchantAccount) stripe_getCheckoutSession(merchantAccount *model.MerchantAccount, sessionID string) (stripe.CheckoutSession, error) {

	const location = "service.MerchantAccount.stripe_getCheckoutSession"

	// Get API Keys from the vault
	restrictedKey, err := service.stripe_getRestrictedKey(merchantAccount)

	if err != nil {
		return stripe.CheckoutSession{}, derp.Wrap(err, location, "Error retrieving API keys")
	}

	// Try to retrieve the Stripe Checkout session
	checkoutSession := stripe.CheckoutSession{}
	txn := remote.Get("https://api.stripe.com/v1/checkout/sessions/"+sessionID).
		Query("expand[]", "customer").
		Query("expand[]", "product").
		With(options.BearerAuth(restrictedKey)).
		Result(&checkoutSession)

	if err := txn.Send(); err != nil {
		return stripe.CheckoutSession{}, derp.Wrap(err, location, "Error connecting to Stripe API")
	}

	return checkoutSession, nil
}

// stripe_getCustomer loads a Customer record from the Stripe API
// https://docs.stripe.com/api/customers/object
func (service *MerchantAccount) stripe_getCustomer(merchantAccount *model.MerchantAccount, customerID string) (stripe.Customer, error) {

	const location = "service.MerchantAccount.stripe_getCustomer"

	// Get API Keys from the vault
	restrictedKey, err := service.stripe_getRestrictedKey(merchantAccount)

	if err != nil {
		return stripe.Customer{}, derp.Wrap(err, location, "Error retrieving API keys")
	}

	customer := stripe.Customer{}
	txn := remote.Get("https://api.stripe.com/v1/customers/" + customerID).
		With(options.BearerAuth(restrictedKey)).
		Result(&customer)

	if err := txn.Send(); err != nil {
		return stripe.Customer{}, derp.Wrap(err, location, "Error connecting to Stripe API")
	}

	return customer, nil
}

// stripe_priceLabel returns a human-friendly label for a Stripe `Price` record.
func (service *MerchantAccount) stripe_priceLabel(price mapof.Any) string {

	// Price in local currency
	result := currency.UnitFormat(price.GetString("currency"), price.GetInt64("unit_amount"))

	// Per recurring interval (if necessary)
	if price.GetString("type") == "recurring" {
		result += " / " + price.GetMap("recurring").GetString("interval")
	}

	// Simply Gorgeous.
	return result
}
