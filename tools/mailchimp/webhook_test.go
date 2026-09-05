package mailchimp

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGetWebhooks reads a Mailchimp response into typed values
func TestGetWebhooks(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {

		require.Equal(t, "/lists/abc123/webhooks", request.URL.Path)

		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"webhooks":[
			{"id":"wh1","url":"https://example.com/.mailchimp/webhook/1"}
		]}`))
	}))

	defer server.Close()

	webhooks, err := testClient(server.URL).GetWebhooks("abc123")

	require.NoError(t, err)
	require.Len(t, webhooks, 1)
	require.Equal(t, "wh1", webhooks[0].ID)
	require.Equal(t, "https://example.com/.mailchimp/webhook/1", webhooks[0].URL)
}

// TestCreateWebhook_NeverRegistersTheApiSource is the guard against an infinite echo loop
func TestCreateWebhook_NeverRegistersTheApiSource(t *testing.T) {

	// Mailchimp fires webhooks for API-initiated changes too, including Emissary's own.
	// Registering `api` would make every member Emissary adds arrive straight back at
	// Emissary's own handler, doubling the User's API traffic against their own quota.

	var body map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {

		require.Equal(t, http.MethodPost, request.Method)

		raw, _ := io.ReadAll(request.Body)
		require.NoError(t, json.Unmarshal(raw, &body))

		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"id":"wh1","url":"https://example.com/hook"}`))
	}))

	defer server.Close()

	events := WebhookEvents{Subscribe: true, Unsubscribe: true, UpEmail: true}
	sources := WebhookSources{User: true, Admin: true}

	webhook, err := testClient(server.URL).CreateWebhook("abc123", "https://example.com/hook", events, sources)

	require.NoError(t, err)
	require.Equal(t, "wh1", webhook.ID)

	sentSources, ok := body["sources"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, sentSources["user"])
	require.Equal(t, true, sentSources["admin"])
	require.Equal(t, false, sentSources["api"], "registering `api` echoes every write back at us (D12)")

	sentEvents, ok := body["events"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, sentEvents["subscribe"])
	require.Equal(t, true, sentEvents["unsubscribe"])
	require.Equal(t, true, sentEvents["upemail"])
	require.Equal(t, false, sentEvents["cleaned"], "bounces are deferred to a separate project")
	require.Equal(t, false, sentEvents["profile"])
	require.Equal(t, false, sentEvents["campaign"])
}

// TestDeleteWebhook confirms the happy path addresses the right resource
func TestDeleteWebhook(t *testing.T) {

	var method, path string

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		method = request.Method
		path = request.URL.Path
		response.WriteHeader(http.StatusNoContent)
	}))

	defer server.Close()

	require.NoError(t, testClient(server.URL).DeleteWebhook("abc123", "wh1"))
	require.Equal(t, http.MethodDelete, method)
	require.Equal(t, "/lists/abc123/webhooks/wh1", path)
}

// TestDeleteWebhook_AlreadyGoneIsSuccess keeps a disconnect from failing over nothing
func TestDeleteWebhook_AlreadyGoneIsSuccess(t *testing.T) {

	// A User can delete this webhook by hand inside Mailchimp.  Refusing to disconnect
	// because there was nothing to disconnect would strand them with a connection they
	// cannot turn off (D37).

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusNotFound)
	}))

	defer server.Close()

	require.NoError(t, testClient(server.URL).DeleteWebhook("abc123", "wh1"))
}

// TestDeleteWebhook_RealFailuresStillFail confirms the 404 tolerance is narrow
func TestDeleteWebhook_RealFailuresStillFail(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusInternalServerError)
	}))

	defer server.Close()

	require.Error(t, testClient(server.URL).DeleteWebhook("abc123", "wh1"))
}
