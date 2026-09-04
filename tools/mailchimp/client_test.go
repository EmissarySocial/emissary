package mailchimp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/benpate/remote"
	"github.com/stretchr/testify/require"
)

// testClient returns a Client pointed at an httptest server instead of Mailchimp
func testClient(serverURL string) Client {

	return Client{
		apiKey:  testHex + "-us6",
		baseURL: serverURL,
		options: []remote.Option{allowLoopback()},
	}
}

// allowLoopback lets a transaction reach an httptest server, which remote's SSRF
// guard blocks by default
func allowLoopback() remote.Option {

	return remote.Option{
		BeforeRequest: func(transaction *remote.Transaction) error {
			transaction.AllowPrivateIPs(true)
			return nil
		},
	}
}

// TestNew_ComposesTheDataCenterAddress confirms a Client addresses the data center it
// was given, and nothing else
func TestNew_ComposesTheDataCenterAddress(t *testing.T) {

	client, err := New(testHex+"-us6", "us21")

	require.NoError(t, err)
	require.Equal(t, "https://us21.api.mailchimp.com/3.0", client.baseURL)
}

// TestNew_RejectsUnusableValues confirms both entry checks run before a Client exists
func TestNew_RejectsUnusableValues(t *testing.T) {

	// A Client is the only thing that can reach the network, so a value that fails an
	// entry check must not survive long enough to become one.

	_, err := New("", "us6")
	require.Error(t, err, "an empty credential must be refused")

	_, err = New(testHex+"-us6", "evil.example.com/x#")
	require.Error(t, err, "a data center that is not a bare DNS label must be refused")

	_, err = New(testHex+"\n-us6", "us6")
	require.Error(t, err, "a credential that cannot travel in a header must be refused")
}

// TestClient_SendsTheCredentialAsABearerToken confirms the credential goes out
// unmodified in the Authorization header
func TestClient_SendsTheCredentialAsABearerToken(t *testing.T) {

	var authorization string

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		authorization = request.Header.Get("Authorization")
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"lists":[]}`))
	}))

	defer server.Close()

	_, err := testClient(server.URL).GetAudiences()

	require.NoError(t, err)
	require.Equal(t, "Bearer "+testHex+"-us6", authorization)
}
