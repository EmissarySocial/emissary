package mailchimp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGetAudiences reads a Mailchimp response into typed values
func TestGetAudiences(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {

		require.Equal(t, "/lists", request.URL.Path)
		require.Equal(t, "1000", request.URL.Query().Get("count"))

		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"lists":[
			{"id":"abc123","name":"Newsletter","stats":{"member_count":42}},
			{"id":"def456","name":"Announcements","stats":{"member_count":0}}
		]}`))
	}))

	defer server.Close()

	audiences, err := testClient(server.URL).GetAudiences()

	require.NoError(t, err)
	require.Len(t, audiences, 2)
	require.Equal(t, "abc123", audiences[0].ID)
	require.Equal(t, "Newsletter", audiences[0].Name)
	require.Equal(t, 42, audiences[0].Stats.MemberCount)
	require.Equal(t, "def456", audiences[1].ID)
}

// TestGetAudiences_Empty confirms an account with no audience is a success, not an error
func TestGetAudiences_Empty(t *testing.T) {

	// Setup only has to prove the credential works. Whether the User has anywhere to sync
	// to is the audience picker's problem, and answering it here would fail a valid key.

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"lists":[]}`))
	}))

	defer server.Close()

	audiences, err := testClient(server.URL).GetAudiences()

	require.NoError(t, err)
	require.Empty(t, audiences)
}

// TestGetAudiences_ErrorsAreActionable pins the mapping from an HTTP status to the
// sentence the User reads on the settings form
func TestGetAudiences_ErrorsAreActionable(t *testing.T) {

	table := []struct {
		name       string
		statusCode int
		contains   string
	}{
		{"a bad key says so", http.StatusUnauthorized, "API key"},
		{"a forbidden key says so", http.StatusForbidden, "API key"},
		{"a bad data center says so", http.StatusNotFound, "data center"},
		{"rate limiting says to wait", http.StatusTooManyRequests, "wait"},
		{"anything else is generic", http.StatusInternalServerError, "audiences"},
	}

	for _, test := range table {

		t.Run(test.name, func(t *testing.T) {

			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				response.Header().Set("Content-Type", "application/json")
				response.WriteHeader(test.statusCode)
				_, _ = response.Write([]byte(`{"title":"Some Mailchimp Problem"}`))
			}))

			defer server.Close()

			_, err := testClient(server.URL).GetAudiences()

			require.Error(t, err)
			require.Contains(t, err.Error(), test.contains)
		})
	}
}

// TestGetAudiences_DoesNotCarryTheCredential guards the promise in README.md that no
// error from this package quotes a key
func TestGetAudiences_DoesNotCarryTheCredential(t *testing.T) {

	// derp redacts credential-bearing headers when it records a transaction, and this
	// package does not return the failed transaction at all. Two independent guards,
	// because one silent regression in either would publish a live key to a log.

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusUnauthorized)
	}))

	defer server.Close()

	client := testClient(server.URL)

	_, err := client.GetAudiences()

	require.Error(t, err)

	// A log entry is the serialized error, so that is what gets searched
	serialized, marshalError := json.Marshal(err)

	require.NoError(t, marshalError)
	require.NotContains(t, string(serialized), client.apiKey)
}
