package stripeapi

import "github.com/benpate/remote"

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
