package mailchimp

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

// TestTagMember confirms the tag call addresses the right member and sends the right body
func TestTagMember(t *testing.T) {

	var method, path string
	var body map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {

		method = request.Method
		path = request.URL.Path

		raw, _ := io.ReadAll(request.Body)
		require.NoError(t, json.Unmarshal(raw, &body))

		// Mailchimp answers this call with 204 and no body at all
		response.WriteHeader(http.StatusNoContent)
	}))

	defer server.Close()

	require.NoError(t, testClient(server.URL).TagMember("abc123", "Sarah@Connor.MIL", "Emissary"))

	require.Equal(t, http.MethodPost, method)
	require.Equal(t, "/lists/abc123/members/"+SubscriberHash("sarah@connor.mil")+"/tags", path)

	tags, ok := body["tags"].([]any)
	require.True(t, ok)
	require.Len(t, tags, 1)

	tag, ok := tags[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "Emissary", tag["name"])
	require.Equal(t, "active", tag["status"], "active is what creates the tag when the audience does not have it yet")
}

// TestTagMember_TrimsTheName keeps a pasted name from becoming a second, near-identical tag
func TestTagMember_TrimsTheName(t *testing.T) {

	var body map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		raw, _ := io.ReadAll(request.Body)
		require.NoError(t, json.Unmarshal(raw, &body))
		response.WriteHeader(http.StatusNoContent)
	}))

	defer server.Close()

	require.NoError(t, testClient(server.URL).TagMember("abc123", "sarah@connor.mil", "  Emissary  "))

	tags := body["tags"].([]any)
	require.Equal(t, "Emissary", tags[0].(map[string]any)["name"])
}

// TestTagMember_RefusesABlankName keeps a request Mailchimp would reject off the wire
func TestTagMember_RefusesABlankName(t *testing.T) {

	requests := 0

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		requests++
		response.WriteHeader(http.StatusNoContent)
	}))

	defer server.Close()

	for _, tag := range []string{"", "   ", "\t\n"} {
		require.Error(t, testClient(server.URL).TagMember("abc123", "sarah@connor.mil", tag))
	}

	require.Zero(t, requests, "a blank tag must never reach Mailchimp")
}

// TestTagMember_KeepsMailchimpsStatusCode confirms the tag call is on the queue path, not the
// form path: the queue reads the status to tell a retry from a permanent failure
func TestTagMember_KeepsMailchimpsStatusCode(t *testing.T) {

	tests := map[string]int{
		"rate limited":     http.StatusTooManyRequests,
		"key revoked":      http.StatusUnauthorized,
		"audience deleted": http.StatusNotFound,
	}

	for name, statusCode := range tests {

		t.Run(name, func(t *testing.T) {

			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				response.WriteHeader(statusCode)
			}))

			defer server.Close()

			err := testClient(server.URL).TagMember("abc123", "sarah@connor.mil", "Emissary")

			require.Error(t, err)
			require.Equal(t, statusCode, derp.ErrorCode(err))
		})
	}
}

// TestTagMember_DoesNotCarryTheCredential guards the leak a failed tag call could spring
func TestTagMember_DoesNotCarryTheCredential(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusUnauthorized)
	}))

	defer server.Close()

	client := testClient(server.URL)

	err := client.TagMember("abc123", "sarah@connor.mil", "Emissary")
	require.Error(t, err)

	// A log entry is the serialized error, so that is what gets searched
	serialized, marshalError := json.Marshal(err)

	require.NoError(t, marshalError)
	require.NotContains(t, string(serialized), client.apiKey)
}
