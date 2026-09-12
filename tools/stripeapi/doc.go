// Package stripeapi is a small, direct client for the Stripe REST API.
//
// It covers the handful of resources Emissary reads -- checkout sessions, customers, prices,
// and subscriptions -- and returns Stripe's own typed values.  Every call takes the
// restricted API key and the connected account id explicitly, so nothing here holds
// credentials or reaches for ambient configuration.
//
// ConnectedAccount produces the remote.Option that scopes a request to a merchant's account
// under Stripe Connect.  Omitting it silently addresses the platform account instead, which
// is why it is a required argument rather than a default.
package stripeapi
