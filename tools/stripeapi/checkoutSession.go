package stripeapi

import (
	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/remote/options"
	"github.com/stripe/stripe-go/v78"
)

// CheckoutSession retrieves a Checkout Session from the Stripe API and returns the customer a slice of priceIDs that were purchased
// in this checkout session.
// https://docs.stripe.com/api/checkout/sessions/object
func CheckoutSession(restrictedKey string, sessionID string) (stripe.CheckoutSession, error) {

	const location = "tools.stripeapi.CheckoutSession"

	// Build a transaction to retrieve the Stripe Checkout session
	checkoutSession := stripe.CheckoutSession{}
	txn := remote.Get("https://api.stripe.com/v1/checkout/sessions/"+sessionID).
		Query("expand[]", "customer").
		Query("expand[]", "line_items").
		Query("expand[]", "subscription").
		With(options.BearerAuth(restrictedKey), options.Debug()).
		Result(&checkoutSession)

	// Send the transaction
	if err := txn.Send(); err != nil {
		return stripe.CheckoutSession{}, derp.Wrap(err, location, "Error connecting to Stripe API")
	}

	// Success
	return checkoutSession, nil
}
