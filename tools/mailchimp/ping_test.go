package mailchimp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestPing confirms the health check reaches the right path with the credential attached
func TestPing(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {

		require.Equal(t, "/ping", request.URL.Path)
		require.NotEmpty(t, request.Header.Get("Authorization"))

		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"health_status":"Everything's Chimpy!"}`))
	}))

	defer server.Close()

	require.NoError(t, testClient(server.URL).Ping())
}

// TestPing_ErrorsAreActionable confirms the four statuses a User can act on each say so
func TestPing_ErrorsAreActionable(t *testing.T) {

	table := []struct {
		name       string
		statusCode int
		contains   string
	}{
		{"a bad key says so", http.StatusUnauthorized, "API key"},
		{"a forbidden key says so", http.StatusForbidden, "API key"},
		{"a bad data center says so", http.StatusNotFound, "data center"},
		{"rate limiting says to wait", http.StatusTooManyRequests, "wait"},
		{"anything else is generic", http.StatusInternalServerError, "Mailchimp"},
	}

	for _, test := range table {

		t.Run(test.name, func(t *testing.T) {

			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				response.Header().Set("Content-Type", "application/json")
				response.WriteHeader(test.statusCode)
				_, _ = response.Write([]byte(`{"title":"Some Mailchimp Problem"}`))
			}))

			defer server.Close()

			err := testClient(server.URL).Ping()

			require.Error(t, err)
			require.Contains(t, err.Error(), test.contains)
		})
	}
}

// TestPing_DoesNotCarryTheCredential guards the one leak a health check could spring
func TestPing_DoesNotCarryTheCredential(t *testing.T) {

	// derp redacts credential-bearing headers when it records a transaction, and this
	// package does not return the failed transaction at all. Two independent guards,
	// because one silent regression in either would publish a live key to a log.

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusUnauthorized)
	}))

	defer server.Close()

	client := testClient(server.URL)

	err := client.Ping()

	require.Error(t, err)

	// A log entry is the serialized error, so that is what gets searched
	serialized, marshalError := json.Marshal(err)

	require.NoError(t, marshalError)
	require.NotContains(t, string(serialized), client.apiKey)
}
