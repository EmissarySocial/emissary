package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/currency"
	api "github.com/EmissarySocial/emissary/tools/stripeapi"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
	"github.com/stripe/stripe-go/v78"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (service *MerchantAccount) stripe_mapSubscriptions(restrictedKey string, userID primitive.ObjectID, subscription *stripe.Subscription) (model.Guest, []model.Purchase, error) {

	const location = "service.MerchantAccount.stripe_mapSubscriptions"

	// NPE check: subscription.Customer must not be null
	if subscription.Customer == nil {
		return model.Guest{}, nil, derp.BadRequestError(location, "Invalid Customer", "The customer value must not be null")
	}

	// NPE check: subscription.Items must not be null
	if subscription.Items == nil {
		return model.Guest{}, nil, derp.BadRequestError(location, "Invalid Subscription", "Stripe Subscription cannot be null")
	}

	// Length Check: must have at least one item in the subscription
	if len(subscription.Items.Data) == 0 {
		return model.Guest{}, nil, derp.BadRequestError(location, "Invalid Subscription", "Sripe Subscription must have at least one item")
	}

	// Load Stripe Customer record from the remote API
	customer, err := api.Customer(restrictedKey, subscription.Customer.ID)

	if err != nil {
		return model.Guest{}, nil, derp.Wrap(err, location, "Error loading customer from Stripe")
	}

	// Load the Guest record that matches the Stripe Customer
	guest, err := service.guestService.LoadOrCreate(customer.Email, model.MerchantAccountTypeStripe, customer.ID)

	if err != nil {
		return model.Guest{}, nil, derp.Wrap(err, location, "Error loading/creating guest by email", customer.Email)
	}

	// Create/Update Purchase records for each "price" line item in the product
	purchases := make(sliceof.Object[model.Purchase], 0, len(subscription.Items.Data))

	for _, item := range subscription.Items.Data {

		// NPTE Check: item
		if item == nil {
			return model.Guest{}, nil, derp.BadRequestError(location, "Invalid Product", "Item cannot be null")
		}

		// NPE Check: item.Price
		if item.Price == nil {
			return model.Guest{}, nil, derp.BadRequestError(location, "Invalid Product", "No price found in product item")
		}

		// Create the new Purchase record
		purchase := model.NewPurchase()
		purchase.UserID = userID
		purchase.GuestID = guest.GuestID

		purchase.RemoteGuestID = guest.RemoteIDs[model.MerchantAccountTypeStripe]
		purchase.RemoteProductID = item.Price.ID
		purchase.RemotePurchaseID = subscription.ID

		purchase.StartDate = subscription.StartDate
		purchase.EndDate = subscription.CurrentPeriodEnd
		purchase.RecurringType = model.PurchaseRecurringTypeOnetime

		switch subscription.Status {

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
				purchase.RecurringType = model.PurchaseRecurringTypeDaily

			case stripe.PriceRecurringIntervalWeek:
				purchase.RecurringType = model.PurchaseRecurringTypeWeekly

			case stripe.PriceRecurringIntervalMonth:
				purchase.RecurringType = model.PurchaseRecurringTypeMonthly

			case stripe.PriceRecurringIntervalYear:
				purchase.RecurringType = model.PurchaseRecurringTypeYearly
			}
		}

		// Append the Purchase to the purchases set
		purchases = append(purchases, purchase)
	}

	// Create/Load the Guest record for this purchase

	// Great success.
	return guest, purchases, nil
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
			Value: merchantAccount.MerchantAccountID.Hex() + ":" + price.ID,
			Label: service.stripe_priceLabel(price),
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

// stripe_recurringType returns the recurring type that matches the provided Stripe `Price`.
func (service *MerchantAccount) stripe_recurringType(price stripe.Price) string {

	if price.Recurring != nil {

		switch price.Recurring.Interval {

		case stripe.PriceRecurringIntervalDay:
			return model.PurchaseRecurringTypeDaily

		case stripe.PriceRecurringIntervalWeek:
			return model.PurchaseRecurringTypeWeekly

		case stripe.PriceRecurringIntervalMonth:
			return model.PurchaseRecurringTypeMonthly

		case stripe.PriceRecurringIntervalYear:
			return model.PurchaseRecurringTypeYearly
		}
	}

	return model.PurchaseRecurringTypeOnetime
}

// stripe_checkoutMode returns the checkout mode that matches the provided Stripe `Price`.
func (service *MerchantAccount) stripe_checkoutMode(price stripe.Price) string {

	if service.stripe_recurringType(price) == model.PurchaseRecurringTypeOnetime {
		return "payment"
	}

	return "subscription"
}
