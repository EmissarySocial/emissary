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
