package camper

import (
	"net/http"

	"github.com/benpate/remote"
	"github.com/benpate/remote/options"
)

// Option is a functional option that modifies a Camper object.  Functional options
// can be applied at creation (via the `New()` function) or afterwards using the
// `.With()` method
type Option func(*Camper)

// RoundTripperMiddleware decorates an http.RoundTripper with additional behavior (caching, logging, etc.)
type RoundTripperMiddleware func(next http.RoundTripper) http.RoundTripper

// WithRemoteOption adds a remote.Option that will be used for all HTTP calls
func WithRemoteOption(option remote.Option) Option {
	return func(camper *Camper) {
		camper.options = append(camper.options, option)
	}
}

// WithRoundTripper specifies a custom HTTP client to use for all remote requests
func WithRoundTripper(roundTripper RoundTripperMiddleware) Option {
	return WithRemoteOption(options.WithRoundTripper(roundTripper))
}

// WithAllowPrivateIPs permits (or forbids) Camper lookups to non-public IP
// addresses. The default is FALSE, so lookups to private/loopback servers are refused.
func WithAllowPrivateIPs(allow bool) Option {

	// The BeforeRequest hook runs before the transport is built, so flipping the
	// transaction flag here reaches the SSRF guard in the dialer.
	return WithRemoteOption(remote.Option{
		BeforeRequest: func(txn *remote.Transaction) error {
			txn.AllowPrivateIPs(allow)
			return nil
		},
	})
}
