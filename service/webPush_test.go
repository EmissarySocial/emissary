package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
	defer response.Body.Close()
	require.Equal(t, http.StatusOK, response.StatusCode)
}
