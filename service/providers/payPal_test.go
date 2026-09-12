package providers

import (
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/paypal"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// PayPal.Refresh requests a token from PayPal whenever the stored one is not valid, so only
// its guard path is exercised here. Connect delegates to Refresh and inherits the same guard.

// TestPayPal_RefreshKeepsAValidToken confirms a token that is still good short-circuits
// Refresh, leaving the stored token untouched and making no request.
func TestPayPal_RefreshKeepsAValidToken(t *testing.T) {

	connection := model.NewConnection()
	token := oauth2.Token{AccessToken: "live-token", Expiry: time.Now().Add(time.Hour)}
	connection.Token = &token

	require.NoError(t, NewPayPal().Refresh(&connection, mapof.NewString()))
	require.Same(t, &token, connection.Token, "the stored token is not replaced")
	require.Equal(t, "live-token", connection.Token.AccessToken)
}

// TestPayPal_ConnectKeepsAValidToken confirms Connect is satisfied by the same guard,
// because it does nothing but delegate to Refresh.
func TestPayPal_ConnectKeepsAValidToken(t *testing.T) {

	connection := model.NewConnection()
	connection.Token = &oauth2.Token{AccessToken: "live-token", Expiry: time.Now().Add(time.Hour)}

	require.NoError(t, NewPayPal().Connect(&connection, mapof.NewString(), "example.com"))
	require.Equal(t, "live-token", connection.Token.AccessToken)
}

// TestPayPal_TokenValidityMatrix pins which stored tokens satisfy the Refresh guard,
// because oauth2.Token.Valid is what decides whether a request goes out.
func TestPayPal_TokenValidityMatrix(t *testing.T) {

	testCases := map[string]struct {
		token         *oauth2.Token
		expectedValid bool
	}{
		"a nil token":              {nil, false},
		"an empty token":           {&oauth2.Token{}, false},
		"no access token":          {&oauth2.Token{Expiry: time.Now().Add(time.Hour)}, false},
		"an expired token":         {&oauth2.Token{AccessToken: "x", Expiry: time.Now().Add(-time.Hour)}, false},
		"a token with no expiry":   {&oauth2.Token{AccessToken: "x"}, true},
		"a token expiring in time": {&oauth2.Token{AccessToken: "x", Expiry: time.Now().Add(time.Hour)}, true},
	}

	for name, testCase := range testCases {

		t.Run(name, func(t *testing.T) {
			assert.Equal(t, testCase.expectedValid, testCase.token.Valid())
		})
	}
}

// TestPayPal_DisconnectClearsTheToken confirms Disconnect drops the credential locally.
// Revoking it at PayPal is still a TODO in the provider.
func TestPayPal_DisconnectClearsTheToken(t *testing.T) {

	connection := model.NewConnection()
	connection.Token = &oauth2.Token{AccessToken: "live-token", Expiry: time.Now().Add(time.Hour)}
	connection.Data.SetString("bnCode", "EMISSARY_SP_PPCP")

	require.NoError(t, NewPayPal().Disconnect(&connection, mapof.NewString()))

	require.Nil(t, connection.Token)
	require.Equal(t, "EMISSARY_SP_PPCP", connection.Data.GetString("bnCode"), "Disconnect keeps the rest of the config")
}

// TestPayPal_DisconnectIsIdempotent confirms a second Disconnect on a cleared token is fine
func TestPayPal_DisconnectIsIdempotent(t *testing.T) {

	connection := model.NewConnection()

	require.NoError(t, NewPayPal().Disconnect(&connection, mapof.NewString()))
	require.NoError(t, NewPayPal().Disconnect(&connection, mapof.NewString()))
	require.Nil(t, connection.Token)
}

// TestPayPal_BeforeSaveIsANoOp confirms BeforeSave writes nothing
func TestPayPal_BeforeSaveIsANoOp(t *testing.T) {

	connection := newTestConnection()
	before := connection

	require.NoError(t, NewPayPal().BeforeSave(&connection, mapof.NewString()))
	require.Equal(t, before, connection)
}

/******************************************
 * Settings Form
 ******************************************/

// TestPayPal_ManualConfigStoresSecretsInTheVault confirms the client ID and secret key are
// vault paths, not data paths, so they are encrypted at rest.
func TestPayPal_ManualConfigStoresSecretsInTheVault(t *testing.T) {

	config := NewPayPal().ManualConfig()

	for _, path := range []string{"vault.clientId", "vault.secretKey"} {

		element, exists := config.Schema.GetElement(path)
		require.True(t, exists, "missing %s", path)

		stringElement, isString := element.(schema.String)
		require.True(t, isString)
		require.True(t, stringElement.Required)
	}

	_, clientIDInData := config.Schema.GetElement("data.clientId")
	require.False(t, clientIDInData, "a credential must never live in data")
}

// TestPayPal_ManualConfigLiveModeEnum confirms the live-mode values match what Refresh reads
func TestPayPal_ManualConfigLiveModeEnum(t *testing.T) {

	config := NewPayPal().ManualConfig()

	element, exists := config.Schema.GetElement("data.liveMode")
	require.True(t, exists)

	stringElement, isString := element.(schema.String)
	require.True(t, isString)
	require.Equal(t, []string{"SANDBOX", "LIVE"}, stringElement.Enum)
}

// TestPayPal_LiveModeSelectsTheAPIHost confirms only the exact string LIVE reaches the
// production API, because Refresh compares liveMode against it directly.
func TestPayPal_LiveModeSelectsTheAPIHost(t *testing.T) {

	testCases := map[string]string{
		"LIVE":     "https://api-m.paypal.com",
		"SANDBOX":  "https://api-m.sandbox.paypal.com",
		"live":     "https://api-m.sandbox.paypal.com",
		"":         "https://api-m.sandbox.paypal.com",
		"anything": "https://api-m.sandbox.paypal.com",
	}

	for liveMode, expectedHost := range testCases {

		t.Run("liveMode="+liveMode, func(t *testing.T) {

			connection := model.NewConnection()
			connection.Data.SetString("liveMode", liveMode)

			host := paypal.APIHost(connection.Data.GetString("liveMode") == "LIVE")
			assert.Equal(t, expectedHost, host)
		})
	}
}

// TestPayPal_IsNotReachableFromTheProviderRegistry pins PayPal as dormant code: there is no
// ConnectionProvider constant for it, so service.Provider.GetProvider can never return it.
func TestPayPal_IsNotReachableFromTheProviderRegistry(t *testing.T) {

	schemaElement, isObject := model.ConnectionSchema().(schema.Object)
	require.True(t, isObject)

	providerID, isString := schemaElement.Properties["providerId"].(schema.String)
	require.True(t, isString)

	for _, value := range providerID.Enum {
		assert.NotContains(t, value, "PAYPAL", "a PayPal provider constant would make this test stale")
		assert.NotContains(t, value, "PAY-PAL")
	}
}
