package service

import (
	"slices"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/currency"
	"github.com/benpate/derp"
	"github.com/benpate/domain"
	"github.com/benpate/form"
	"github.com/benpate/remote"
	"github.com/benpate/remote/options"
	"github.com/benpate/rosetta/compare"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

		// Configure a new Webhook in the Stripe API
		webhookResult := mapof.NewAny()
		txn := remote.Post("https://api.stripe.com/v1/webhook_endpoints").
			With(options.BearerAuth(restrictedKey)).
			Query("url", service.host+"/.stripe/webhook").
			Query("description", domain.NameOnly(service.host)+" supscription updates").
			Query("enabled_events[]", "checkout.session.completed").
			Query("enabled_events[]", "customer.subscription.created").
			Query("enabled_events[]", "customer.subscription.deleted").
			Query("enabled_events[]", "customer.subscription.paused").
			Query("enabled_events[]", "customer.subscription.resumed").
			Query("enabled_events[]", "customer.subscription.updated").
			Result(&webhookResult)

		if err := txn.Send(); err != nil {
			return derp.Wrap(err, location, "Error connecting to Stripe API")
		}

		// Save the webhook ID into the MerchantAccount
		merchantAccount.Plaintext.SetString("webhook", webhookResult.GetString("id"))
	}

	return nil
}

// stripe_refreshSubscription refreshes the subscription data for a Stripe subscription
func (service *MerchantAccount) stripe_refreshSubscription(merchantAccount *model.MerchantAccount, subscription *model.Subscription) error {

	const location = "service.MerchantAccount.stripe_refreshSubscription"

	// Get API Keys from the vault
	restrictedKey, err := service.stripe_getRestrictedKey(merchantAccount)

	if err != nil {
		return derp.Wrap(err, location, "Error retrieving API keys")
	}

	// Load the associated Stripe `Price` record
	price := mapof.NewAny()
	txn := remote.Get("https://api.stripe.com/v1/prices/" + subscription.SubscriptionToken).
		With(options.BearerAuth(restrictedKey)).
		Result(&price)

	if err := txn.Send(); err != nil {
		return derp.Wrap(err, location, "Error connecting to Stripe API")
	}

	// Set/Update the RecurringType for the subscription
	switch price.GetMap("recurring").GetString("interval") {
	case "day":
		subscription.RecurringType = model.SubscriptionRecurringTypeDaily
	case "week":
		subscription.RecurringType = model.SubscriptionRecurringTypeWeekly
	case "month":
		subscription.RecurringType = model.SubscriptionRecurringTypeMonthly
	case "year":
		subscription.RecurringType = model.SubscriptionRecurringTypeYearly
	default:
		subscription.RecurringType = model.SubscriptionRecurringTypeOnetime
	}

	// Set/Update the Price label for the subscription
	subscription.Price = service.stripe_priceLabel(price)

	// Subbess.
	return nil
}

// stripe_getPrices retrieves all prices from the Stripe API and returns them as a list of LookupCodes
func (service *MerchantAccount) stripe_getPrices(merchantAccount *model.MerchantAccount) ([]form.LookupCode, error) {

	const location = "service.MerchantAccount.paypal_getSubscriptions"

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

func (service *MerchantAccount) stripe_getCheckoutURL(merchantAccount *model.MerchantAccount, subscription *model.Subscription, successURL string, cancelURL string) (string, error) {

	const location = "service.MerchantAccount.stripe_getCheckoutURL"
	restrictedKey, err := service.stripe_getRestrictedKey(merchantAccount)

	if err != nil {
		return "", derp.Wrap(err, location, "Error retrieving restricted key")
	}

	// Send checkout session to the Stripe API
	checkoutResult := mapof.NewAny()
	transactionID := primitive.NewObjectID().Hex()

	txn := remote.Post("https://api.stripe.com/v1/checkout/sessions").
		With(options.BearerAuth(restrictedKey)).
		ContentType("application/x-www-form-urlencoded").
		Form("mode", iif((subscription.RecurringType == model.SubscriptionRecurringTypeOnetime), "payment", "recurring")).
		Form("line_items[0][price]", subscription.SubscriptionToken).
		Form("line_items[0][quantity]", "1").
		Form("ui_mode", "hosted").
		Form("client_reference_id", transactionID).
		Form("cancel_url", cancelURL).
		Form("success_url", successURL+"?trasactionId="+transactionID).
		Result(&checkoutResult)

	if err := txn.Send(); err != nil {
		return "", derp.Wrap(err, location, "Error connecting to Stripe API")
	}

	// Return the URL to the caller
	return checkoutResult.GetString("url"), nil
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
