package model

import "time"

// https://swicg.github.io/activitypub-data-portability/lola#Authorization
const OAuthUserTokenScopeActivityPubPortability = "activitypub_account_portability"

// OAuth token lifecycle durations (see emissary-specs/OAUTH-REFRESH-TOKENS.md).
const (
	// OAuthAccessTokenLifetime is how long an issued access-token JWT is valid.
	// The client refreshes before this elapses. Not configurable by design.
	OAuthAccessTokenLifetime = 1 * time.Hour

	// OAuthRefreshGracePeriod is the window after a rotation during which the
	// immediately-prior refresh secret is still tolerated (absorbs a client that
	// lost the token-endpoint response and retried). It is also the theft-tolerance
	// gap for a stolen prior secret, so it is kept tight.
	OAuthRefreshGracePeriod = 5 * time.Minute

	// OAuthCodeLifetime is how long an authorization code may be exchanged after it
	// is issued. Codes are single-use AND short-lived (RFC 6749 §4.1.2).
	OAuthCodeLifetime = 10 * time.Minute
)
