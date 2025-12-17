package stripeapi

import (
	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/remote/options"
	"github.com/stripe/stripe-go/v78"
)

// Customer loads a Customer record from the Stripe API
// https://docs.stripe.com/api/customers/object
func Customer(restrictedKey string, connectedAccountID string, customerID string) (stripe.Customer, error) {

	const location = "tools.stripeapi.Customer"

	// Build a transaction to retrieve the Stripe Customer
	customer := stripe.Customer{}
	txn := remote.Get("https://api.stripe.com/v1/customers/" + customerID).
		With(options.BearerAuth(restrictedKey)).
		With(ConnectedAccount(connectedAccountID)).
		Result(&customer)

	// Send the transaction
	if err := txn.Send(); err != nil {
		return stripe.Customer{}, derp.Wrap(err, location, "Error connecting to Stripe API", derp.WithInternalError())
	}

	// Success
	return customer, nil
}
