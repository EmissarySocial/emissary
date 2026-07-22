package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

// TestWebPush_EndpointIsAllowed_Production confirms the registration guard rejects internal
// endpoints (the SSRF targets from the report) while permitting real push-service hosts.
func TestWebPush_EndpointIsAllowed_Production(t *testing.T) {

	service := &WebPush{allowPrivateIPs: false}

	// Internal / SSRF targets must be rejected.
	require.False(t, service.EndpointIsAllowed("http://169.254.169.254/latest/meta-data/"))
	require.False(t, service.EndpointIsAllowed("http://127.0.0.1:8080/@me"))
	require.False(t, service.EndpointIsAllowed("http://localhost/x"))
	require.False(t, service.EndpointIsAllowed("http://10.0.0.5:27017/"))
	require.False(t, service.EndpointIsAllowed("http://mongo.internal/"))

	// Real push-service endpoints must be allowed.
	require.True(t, service.EndpointIsAllowed("https://fcm.googleapis.com/fcm/send/abc123"))
	require.True(t, service.EndpointIsAllowed("https://updates.push.services.mozilla.com/wpush/v2/xyz"))
}

// TestWebPush_EndpointIsAllowed_DevAllowsAll confirms a local/dev instance keeps its ability to
// reach private addresses (so it can talk to itself), matching ActivityStream.AllowPrivateIPs().
func TestWebPush_EndpointIsAllowed_DevAllowsAll(t *testing.T) {

	service := &WebPush{allowPrivateIPs: true}

	require.True(t, service.EndpointIsAllowed("http://169.254.169.254/latest/meta-data/"))
	require.True(t, service.EndpointIsAllowed("http://127.0.0.1:8080/@me"))
}

// TestVapidSubscriber_IsAlwaysBare confirms every value this function can return is a bare address
//
// webpush-go prepends "mailto:" to any subscriber that does not begin with "https:", so a "mailto:"
// URI double-prefixes inside the signed token and the push service rejects the whole JWT.
func TestVapidSubscriber_IsAlwaysBare(t *testing.T) {

	subscribers := []string{
		vapidSubscriber("owner@example.social", "example.social"), // configured owner
		vapidSubscriber("", "example.social"),                     // derived from hostname
		vapidSubscriber("", "127.0.0.1"),                          // fallback
		vapidSubscriber("not-an-email", "127.0.0.1"),              // invalid owner -> fallback
	}

	for _, subscriber := range subscribers {
		require.NotContains(t, subscriber, "mailto:", "VAPID subscriber must be BARE -- webpush-go adds the scheme")
	}
}

// TestVapidSubscriber_PrefersOwnerEmail confirms the Domain owner's real address wins
func TestVapidSubscriber_PrefersOwnerEmail(t *testing.T) {

	require.Equal(t, "owner@example.social", vapidSubscriber("owner@example.social", "example.social"))

	// Even on a local/dev host, a real configured owner beats the placeholder.
	require.Equal(t, "owner@example.social", vapidSubscriber("owner@example.social", "127.0.0.1"))
}

// TestVapidSubscriber_RejectsUnusableOwnerEmail confirms an empty, malformed, or
// display-name-wrapped owner address falls back instead of being forwarded
func TestVapidSubscriber_RejectsUnusableOwnerEmail(t *testing.T) {

	require.Equal(t, vapidSubscriberFallback, vapidSubscriber("", "127.0.0.1"))
	require.Equal(t, vapidSubscriberFallback, vapidSubscriber("not-an-email", "127.0.0.1"))
	require.Equal(t, vapidSubscriberFallback, vapidSubscriber("Ben <ben@example.social>", "127.0.0.1"))
}

// TestVapidSubscriber_LocalHostsFallBack confirms a local/dev host yields the placeholder rather
// than a contact nobody could reach
func TestVapidSubscriber_LocalHostsFallBack(t *testing.T) {

	require.Equal(t, vapidSubscriberFallback, vapidSubscriber("", "127.0.0.1"))
	require.Equal(t, vapidSubscriberFallback, vapidSubscriber("", "127.0.0.1:8080"))
	require.Equal(t, vapidSubscriberFallback, vapidSubscriber("", "localhost"))
	require.Equal(t, vapidSubscriberFallback, vapidSubscriber("", "localhost:8080"))
	require.Equal(t, vapidSubscriberFallback, vapidSubscriber("", "http://localhost:8080"))

	// A dotless host cannot be an email domain either.
	require.Equal(t, vapidSubscriberFallback, vapidSubscriber("", "intranet"))
}

// TestVapidSubscriber_DerivedFromHostname confirms that with no configured owner, a production
// hostname still yields a plausible contact, and that a port never survives into it
func TestVapidSubscriber_DerivedFromHostname(t *testing.T) {

	require.Equal(t, "admin@example.social", vapidSubscriber("", "example.social"))
	require.Equal(t, "admin@example.social", vapidSubscriber("", "example.social:8443"))
	require.Equal(t, "admin@example.social", vapidSubscriber("", "https://example.social/"))
}

// TestWebPushHTTPClient_BlocksLoopback confirms the delivery-time guard: the production client
// refuses to connect to a loopback address (the authoritative, rebinding-safe backstop), while the
// dev client connects normally.
func TestWebPushHTTPClient_BlocksLoopback(t *testing.T) {

	// A loopback httptest server stands in for any internal target.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// The guarded (production) client must refuse to connect to the loopback address.
	guarded := webPushHTTPClient(false)
	_, err := guarded.Get(server.URL)
	require.Error(t, err)

	// The unguarded (dev) client connects normally.
	dev := webPushHTTPClient(true)
	response, err := dev.Get(server.URL)
	require.Nil(t, err)
	defer derp.ReportFunc(response.Body.Close)

	require.Equal(t, http.StatusOK, response.StatusCode)
}
