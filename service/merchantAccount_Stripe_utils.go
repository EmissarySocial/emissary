package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/currency"
	api "github.com/EmissarySocial/emissary/tools/stripeapi"
	"github.com/benpate/derp"
	"github.com/stripe/stripe-go/v78"
)

// stripe_getRestrictedKey retrieves the restricted API key for the specified MerchantAccount
func (service *MerchantAccount) stripe_getRestrictedKey(merchantAccount *model.MerchantAccount) (string, error) {

	const location = "service.MerchantAccount.stripe_getRestrictedKey"

	if merchantAccount == nil {
		return "", derp.InternalError(location, "MerchantAccount cannot be nil")
	}

	apiKeys, err := service.DecryptVault(merchantAccount, "restrictedKey")

	if err != nil {
		return "", derp.Wrap(err, location, "Error retrieving API keys")
	}

	return apiKeys.GetString("restrictedKey"), nil
}

// stripe_getConnectedAccountID retrieves the connected account ID for the specified MerchantAccount
// This is ONLY used by Stripe Connect accounts, to retrieve a specific merchant that we're working with.
// For all others (e.g. Stripe) this will return an empty string.
func (service *MerchantAccount) stripe_getConnectedAccountID(merchantAccount *model.MerchantAccount) string {

	if merchantAccount == nil {
		return ""
	}

	if merchantAccount.Type != model.ConnectionProviderStripeConnect {
		return ""
	}

	return merchantAccount.Plaintext.GetString("accountId")
}

func (service *MerchantAccount) stripe_getPrices(merchantAccount *model.MerchantAccount, priceIDs ...string) ([]model.Product, error) {

	const location = "service.MerchantAccount.stripe_getPrices"

	// Retrieve the restricted key for this Merchant Account
	restrictedKey, err := service.stripe_getRestrictedKey(merchantAccount)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error retrieving restricted key")
	}

	connectedAccountID := merchantAccount.Plaintext.GetString("accountId")

	// Load the Prices from the Stripe API
	prices, err := api.Prices(restrictedKey, connectedAccountID, priceIDs...)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error retrieving prices from Stripe")
	}

	result := make([]model.Product, 0, len(prices))

	for _, price := range prices {

		// RULE: Price must be active
		if !price.Active {
			continue
		}

		// RULE: Product must not be nil
		if price.Product == nil {
			continue
		}

		// RULE: Product must be active
		if !price.Product.Active {
			continue
		}

		// Append the model.Product to the result
		product := model.NewProduct()
		product.MerchantAccountID = merchantAccount.MerchantAccountID
		product.UserID = merchantAccount.UserID
		product.RemoteID = price.ID
		product.Name = price.Product.Name
		product.Price = service.stripe_priceLabel(price)
		product.Icon = "stripe"
		product.AdminHref = "https://dashboard.stripe.com/prices/" + price.ID

		result = append(result, product)
	}

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

// stripe_recurringType returns the INTERNAL recurring type constant that matches the provided Stripe `Price` record.
func (service *MerchantAccount) stripe_recurringType(price stripe.Price) string {

	if price.Recurring != nil {

		switch price.Recurring.Interval {

		case stripe.PriceRecurringIntervalDay:
			return model.PrivilegeRecurringTypeDay

		case stripe.PriceRecurringIntervalWeek:
			return model.PrivilegeRecurringTypeWeek

		case stripe.PriceRecurringIntervalMonth:
			return model.PrivilegeRecurringTypeMonth

		case stripe.PriceRecurringIntervalYear:
			return model.PrivilegeRecurringTypeYear
		}
	}

	return model.PrivilegeRecurringTypeOnetime
}

// stripe_checkoutMode returns the checkout mode that matches the provided Stripe `Price` (either "payment" or "subscription")
func (service *MerchantAccount) stripe_checkoutMode(price stripe.Price) string {

	if price.Recurring == nil {
		return "payment"
	}

	return "subscription"
}
