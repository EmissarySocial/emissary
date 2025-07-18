package stripeapi

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/remote/options"
	"github.com/stripe/stripe-go/v78"
)

// Subscription loads a Subscription record from the Stripe API
// https://docs.stripe.com/api/subscriptions/object
func Subscription(restrictedKey string, connectedAccountID string, subscriptionID string) (stripe.Subscription, error) {

	const location = "tools.stripeapi.Subscription"

	// Build a transaction to retrieve the Stripe Subscription
	subscription := stripe.Subscription{}
	txn := remote.Get("https://api.stripe.com/v1/subscriptions/" + subscriptionID).
		With(options.BearerAuth(restrictedKey)).
		With(ConnectedAccount(connectedAccountID)).
		Result(&subscription)

	// Send the transaction
	if err := txn.Send(); err != nil {
		return stripe.Subscription{}, derp.Wrap(err, location, "Error connecting to Stripe API", derp.WithCode(http.StatusInternalServerError))
	}

	// Success
	return subscription, nil
}

// SubscriptionIsActive returns true if the subscription is active or trialing
func SubscriptionIsActive(subscription stripe.Subscription) bool {

	switch subscription.Status {

	case stripe.SubscriptionStatusActive:
		return true

	case stripe.SubscriptionStatusTrialing:
		return true

	}

	return false
}

func SubscriptionCancel(restrictedKey string, connectedAccountID string, subscriptionID string) error {

	const location = "tools.stripeapi.SubscriptionCancel"

	txn := remote.Delete("https://api.stripe.com/v1/subscriptions/" + subscriptionID).
		With(options.BearerAuth(restrictedKey)).
		With(ConnectedAccount(connectedAccountID))

	if err := txn.Send(); err != nil {
		return derp.Wrap(err, location, "Error canceling subscription", derp.WithCode(http.StatusInternalServerError))
	}

	return nil
}
