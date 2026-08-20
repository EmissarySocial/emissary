package stripeapi

import "github.com/benpate/remote"

// ConnectedAccount returns a remote.Option that signs a request on behalf of a connected Stripe account
func ConnectedAccount(connectedAccountID string) remote.Option {

	return remote.Option{

		BeforeRequest: func(txn *remote.Transaction) error {
			if connectedAccountID != "" {
				txn.Header("Stripe-Account", connectedAccountID)
			}
			return nil
		},
	}
}
