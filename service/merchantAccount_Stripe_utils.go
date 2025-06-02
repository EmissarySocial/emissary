package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/currency"
	api "github.com/EmissarySocial/emissary/tools/stripeapi"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/slice"
	"github.com/stripe/stripe-go/v78"
)

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
		return "", derp.Wrap(err, location, "Error retrieving API keys", propertyName)
	}

	return apiKeys.GetString(propertyName), nil
}

func (service *MerchantAccount) stripe_getPrices(merchantAccount *model.MerchantAccount, priceIDs ...string) ([]form.LookupCode, error) {

	const location = "service.MerchantAccount.stripe_getPrices"

	// Retrieve the restricted key for this Merchant Account
	restrictedKey, err := service.stripe_getRestrictedKey(merchantAccount)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error retrieving restricted key")
	}

	// Load the Prices from the Stripe API
	prices, err := api.Prices(restrictedKey, priceIDs...)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error retrieving prices from Stripe")
	}

	result := slice.Map(prices, func(price stripe.Price) form.LookupCode {
		return form.LookupCode{
			Group: price.Product.Name,
			Label: service.stripe_priceLabel(price),
			Value: merchantAccount.MerchantAccountID.Hex() + ":" + price.ID,
		}
	})

	return result, nil
}

// stripe_priceLabel returns a human-friendly label for a Stripe `Price` record.
func (service *MerchantAccount) stripe_priceLabel(price stripe.Price) string {

	// Price in local currency
	result := currency.UnitFormat(string(price.Currency), price.UnitAmount)

	// Per recurring interval (if necessary)
	if price.Type == "recurring" {
		if recurring := price.Recurring; recurring != nil {
			result += " / " + string(recurring.Interval)
		}
	}

	// Simply Gorgeous.
	return result
}

// stripe_checkoutMode returns the checkout mode that matches the provided Stripe `Price`.
func (service *MerchantAccount) stripe_checkoutMode(price stripe.Price) string {

	if price.Recurring == nil {
		return "payment"
	}

	return "subscription"
}
