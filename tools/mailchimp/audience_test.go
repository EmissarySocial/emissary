package mailchimp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGetAudience reads one audience by the ID a User pasted
func TestGetAudience(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {

		require.Equal(t, "/lists/abc123", request.URL.Path)

		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"id":"abc123","name":"Newsletter","stats":{"member_count":412}}`))
	}))

	defer server.Close()

	audience, err := testClient(server.URL).GetAudience("abc123")

	require.NoError(t, err)
	require.Equal(t, "abc123", audience.ID)
	require.Equal(t, "Newsletter", audience.Name)
	require.Equal(t, 412, audience.Stats.MemberCount)
}

// TestGetAudience_NotFoundBlamesTheIdNotTheDataCenter is the whole reason this call does not
// share describeError
func TestGetAudience_NotFoundBlamesTheIdNotTheDataCenter(t *testing.T) {

	// The shared 404 message says "Mailchimp has no account in this data center", which sends
	// the User off to re-check a value that was already correct.  A 404 HERE means the pasted
	// Audience ID -- most often because they copied the number out of their address bar,
	// which is a different identifier that the API will never accept.

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusNotFound)
	}))

	defer server.Close()

	_, err := testClient(server.URL).GetAudience("not-a-real-id")

	require.Error(t, err)
	require.Contains(t, err.Error(), "audience")
	require.Contains(t, err.Error(), "address bar")
	require.NotContains(t, err.Error(), "data center")
}

// TestGetAudience_OtherFailuresKeepTheSharedMessages confirms the override is narrow
func TestGetAudience_OtherFailuresKeepTheSharedMessages(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusUnauthorized)
	}))

	defer server.Close()

	_, err := testClient(server.URL).GetAudience("abc123")

	require.Error(t, err)
	require.Contains(t, err.Error(), "API key")
}
