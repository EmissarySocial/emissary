// Package stripe handles inbound webhooks from Stripe for the hosted signup flow.
//
// The single route here receives Stripe's notification that a signup checkout completed, and
// provisions the resulting Domain.  It is separate from handler/stripe.go in the parent
// package, which handles per-merchant subscription webhooks.
//
// The route is public and unauthenticated, so the request's signature is the only thing that
// makes it trustworthy: verify it against the endpoint secret and bound the body before
// reading it.  Nothing in the payload may be treated as authenticated fact on its own.
package stripe
