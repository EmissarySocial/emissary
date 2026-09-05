package mailchimp

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

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
	require.Contains(t, err.Error(), "API key")

	require.Error(t, client.UnsubscribeMember("abc123", "sarah@connor.mil"))
}
