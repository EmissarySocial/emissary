package mailchimp

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

// TestSubscriberHash pins the addressing scheme Mailchimp requires
func TestSubscriberHash(t *testing.T) {

	// Mailchimp will not accept a member addressed any other way, and gets this wrong
	// silently: a mis-cased hash simply names a member who does not exist.

	// Computed independently (`printf 'sarah@connor.mil' | md5`), not copied from this code
	expected := "6a48f537bd8150a8bf60538994919049"

	require.Equal(t, expected, SubscriberHash("sarah@connor.mil"))
	require.Equal(t, expected, SubscriberHash("Sarah@Connor.MIL"), "the address is lowercased first")
	require.Equal(t, expected, SubscriberHash("  sarah@connor.mil  "), "and trimmed")
}

// TestSetMember confirms the upsert addresses the right member and sends the right body
func TestSetMember(t *testing.T) {

	var method, path string
	var body map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {

		method = request.Method
		path = request.URL.Path

		raw, _ := io.ReadAll(request.Body)
		require.NoError(t, json.Unmarshal(raw, &body))

		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{}`))
	}))

	defer server.Close()

	member := Member{
		EmailAddress: "sarah@connor.mil",
		Status:       MemberStatusSubscribed,
		MergeFields:  map[string]string{"FNAME": "Sarah", "LNAME": "Connor", "EMISSARYID": "abc"},
		IPSignup:     "10.0.0.1",
	}

	require.NoError(t, testClient(server.URL).SetMember("abc123", member))

	// PUT is what makes this an upsert, and an upsert is what lets Follower.Save call it on
	// every save without caring whether the member is already there.
	require.Equal(t, http.MethodPut, method)
	require.Equal(t, "/lists/abc123/members/"+SubscriberHash("sarah@connor.mil"), path)

	require.Equal(t, "sarah@connor.mil", body["email_address"])
	require.Equal(t, "subscribed", body["status"])
	require.Equal(t, "10.0.0.1", body["ip_signup"])

	merges, ok := body["merge_fields"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "Sarah", merges["FNAME"])
	require.Equal(t, "Connor", merges["LNAME"])
	require.Equal(t, "abc", merges["EMISSARYID"])
}

// TestSetMember_OmitsEmptyOptionalFields keeps Emissary from overwriting Mailchimp's data
// with blanks
func TestSetMember_OmitsEmptyOptionalFields(t *testing.T) {

	// A Follower who signed up before the IP was captured has none, permanently. Sending an
	// empty string would write that blank over whatever Mailchimp already holds.

	var body map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		raw, _ := io.ReadAll(request.Body)
		require.NoError(t, json.Unmarshal(raw, &body))
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{}`))
	}))

	defer server.Close()

	member := Member{EmailAddress: "sarah@connor.mil", Status: MemberStatusSubscribed}

	require.NoError(t, testClient(server.URL).SetMember("abc123", member))

	require.NotContains(t, body, "ip_signup")
	require.NotContains(t, body, "merge_fields")
}

// TestUnsubscribeMember confirms the status change addresses the right member
func TestUnsubscribeMember(t *testing.T) {

	var method, path string
	var body map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {

		method = request.Method
		path = request.URL.Path

		raw, _ := io.ReadAll(request.Body)
		require.NoError(t, json.Unmarshal(raw, &body))

		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{}`))
	}))

	defer server.Close()

	require.NoError(t, testClient(server.URL).UnsubscribeMember("abc123", "sarah@connor.mil"))

	require.Equal(t, http.MethodPatch, method)
	require.Equal(t, "/lists/abc123/members/"+SubscriberHash("sarah@connor.mil"), path)
	require.Equal(t, "unsubscribed", body["status"])
}

// TestUnsubscribeMember_UnknownAddressIsSuccess keeps a task from retrying forever
func TestUnsubscribeMember_UnknownAddressIsSuccess(t *testing.T) {

	// Emissary pushes only on confirmation (D5), so someone who unsubscribes before their
	// first push was never a member here. A 404 is the outcome, not a failure.

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusNotFound)
	}))

	defer server.Close()

	require.NoError(t, testClient(server.URL).UnsubscribeMember("abc123", "sarah@connor.mil"))
}

// TestMember_ErrorsAreActionable confirms a real failure still reports
func TestMember_ErrorsAreActionable(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusUnauthorized)
	}))

	defer server.Close()

	client := testClient(server.URL)

	err := client.SetMember("abc123", Member{EmailAddress: "sarah@connor.mil"})
	require.Error(t, err)

	require.Error(t, client.UnsubscribeMember("abc123", "sarah@connor.mil"))
}

// TestMember_KeepsMailchimpsStatusCode is the regression guard for three silent failures
func TestMember_KeepsMailchimpsStatusCode(t *testing.T) {

	// A member call is never shown to a User -- it runs in a queue task. What reads its
	// error is `requeue`, which tells a retry from a permanent failure by status code, and
	// `mailchimp_reportMemberError`, which flags a revoked key by status code. Routing this
	// path through `describeError` rewrote every one of them to 422, which silently made a
	// rate limit permanent and made a rejected credential undetectable.

	tests := map[string]int{
		"rate limited":      http.StatusTooManyRequests,
		"key revoked":       http.StatusUnauthorized,
		"key lacks access":  http.StatusForbidden,
		"audience deleted":  http.StatusNotFound,
		"mailchimp is down": http.StatusInternalServerError,
	}

	for name, statusCode := range tests {

		t.Run(name, func(t *testing.T) {

			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				response.WriteHeader(statusCode)
			}))

			defer server.Close()

			err := testClient(server.URL).SetMember("abc123", Member{EmailAddress: "sarah@connor.mil"})

			require.Error(t, err)
			require.Equal(t, statusCode, derp.ErrorCode(err))
		})
	}
}

// TestMember_RateLimitIsRetryable pins the classification that keeps a subscriber from
// being silently dropped
func TestMember_RateLimitIsRetryable(t *testing.T) {

	// A bulk sync is exactly what provokes a 429, so this is the busiest path, not the
	// rarest one. As a 422 it read as a client error, and the queue gave up for good.

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusTooManyRequests)
	}))

	defer server.Close()

	err := testClient(server.URL).SetMember("abc123", Member{EmailAddress: "sarah@connor.mil"})

	require.Error(t, err)

	isTooMany, delay := derp.IsTooManyRequests(err)
	require.True(t, isTooMany, "a rate limit must be retryable, not a permanent failure")
	require.Greater(t, delay, time.Duration(0))
}

// TestMember_RejectedCredentialIsDetectable pins the signal that flags a connection for
// reconnection
func TestMember_RejectedCredentialIsDetectable(t *testing.T) {

	// This is what tells the User their key stopped working. As a 422 the check never
	// fired, so the connection stayed READY and every push failed in silence.

	for name, statusCode := range map[string]int{"unauthorized": http.StatusUnauthorized, "forbidden": http.StatusForbidden} {

		t.Run(name, func(t *testing.T) {

			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				response.WriteHeader(statusCode)
			}))

			defer server.Close()

			err := testClient(server.URL).SetMember("abc123", Member{EmailAddress: "sarah@connor.mil"})

			require.Error(t, err)
			require.True(t, derp.IsUnauthorized(err) || derp.IsForbidden(err))
		})
	}
}

// TestMember_TransportFailureIsRetryable confirms a network error is not mistaken for a
// permanent client error
func TestMember_TransportFailureIsRetryable(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	server.Close() // Nothing is listening, so the request cannot complete

	err := testClient(server.URL).SetMember("abc123", Member{EmailAddress: "sarah@connor.mil"})

	require.Error(t, err)
	require.True(t, derp.IsServerError(err), "a transport failure must be retryable")
}
