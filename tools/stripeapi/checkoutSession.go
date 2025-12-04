package stripeapi

import (
	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/remote/options"
	"github.com/benpate/rosetta/sliceof"
	"github.com/stripe/stripe-go/v78"
)

// CheckoutSession retrieves a Checkout Session from the Stripe API and returns the customer a slice of priceIDs that were purchased
// in this checkout session.
// https://docs.stripe.com/api/checkout/sessions/object
func CheckoutSession(restrictedKey string, connectedAccountID string, sessionID string) (stripe.CheckoutSession, error) {

	const location = "tools.stripeapi.CheckoutSession"

	// Build a transaction to retrieve the Stripe Checkout session
	checkoutSession := stripe.CheckoutSession{}
	txn := remote.Get("https://api.stripe.com/v1/checkout/sessions/"+sessionID).
		With(options.BearerAuth(restrictedKey), options.Debug()).
		With(ConnectedAccount(connectedAccountID)).
		Query("expand[]", "customer").
		Query("expand[]", "line_items").
		Query("expand[]", "subscription").
		Result(&checkoutSession)

	// Send the transaction
	if err := txn.Send(); err != nil {
		return stripe.CheckoutSession{}, derp.Wrap(err, location, "Error connecting to Stripe API", derp.WithInternalError())
	}

	// Success
	return checkoutSession, nil
}

// CheckoutSessionProductIDs safely retrieves a slice of product IDs from Stripe Checkout Session.
func CheckoutSessionProductIDs(checkoutSession stripe.CheckoutSession) sliceof.String {

	// NPE Checks
	if checkoutSession.LineItems == nil || checkoutSession.LineItems.Data == nil {
		return sliceof.NewString()
	}

	result := sliceof.NewString()

	// Collect the product IDs from the line items
	for _, lineItem := range checkoutSession.LineItems.Data {

		// NPE Checks
		if lineItem == nil || lineItem.Price == nil || lineItem.Price.Product == nil {
			continue
		}

		// Append the product ID to the result
		result = append(result, lineItem.Price.Product.ID)
	}

	// Done.
	return result
}
