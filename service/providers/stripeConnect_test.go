package providers

import (
	"net/url"
	"regexp"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// StripeConnect.Connect registers a webhook with the Stripe API unless one of its two guards
// fires first. Only those guard paths are exercised here.

// TestStripeConnect_ConnectSkipsLocalHostnames confirms a non-public host returns early,
// because Stripe cannot reach a webhook URL that does not resolve on the internet.
func TestStripeConnect_ConnectSkipsLocalHostnames(t *testing.T) {

	localHosts := []string{
		"localhost",
		"localhost:8080",
		"https://localhost",
		"127.0.0.1",
		"127.0.0.1:443",
		"::1",
		"[::1]:443",
		"10.0.0.1",
		"192.168.1.50",
		"172.16.4.9",
		"emissary.local",
		"printer.local.",
		"host.docker.internal",
		"my-cluster.internal",
	}

	for _, host := range localHosts {

		t.Run(host, func(t *testing.T) {

			connection := model.NewConnection()

			require.NoError(t, NewStripeConnect().Connect(&connection, mapof.NewString(), host))
			assert.Empty(t, connection.Data.GetString("webhook"), "no webhook is registered")
			assert.False(t, connection.Vault.HasString("webhookSecret"))
		})
	}
}

// TestStripeConnect_ConnectSkipsAnExistingWebhook confirms a Connection that already carries
// a webhook ID is left alone, so a re-save does not pile up duplicate webhooks at Stripe.
func TestStripeConnect_ConnectSkipsAnExistingWebhook(t *testing.T) {

	connection := model.NewConnection()
	connection.Data.SetString("webhook", "we_1PabcdEFGH")

	require.NoError(t, NewStripeConnect().Connect(&connection, mapof.NewString(), "example.com"))
	require.Equal(t, "we_1PabcdEFGH", connection.Data.GetString("webhook"), "the existing webhook is preserved")
}

// TestStripeConnect_ConnectChecksTheHostBeforeTheWebhook pins the guard order: a local host
// returns early even when no webhook exists, which is the only path a dev server ever takes.
func TestStripeConnect_ConnectChecksTheHostBeforeTheWebhook(t *testing.T) {

	connection := model.NewConnection()
	require.Empty(t, connection.Data.GetString("webhook"))

	require.NoError(t, NewStripeConnect().Connect(&connection, mapof.NewString(), "localhost"))
	require.Empty(t, connection.Data.GetString("webhook"))
}

// TestStripeConnect_ConnectAcceptsANilVaultOnAGuardPath confirms the restricted key is only
// read after both guards pass, so a guarded call never touches the vault.
func TestStripeConnect_ConnectAcceptsANilVaultOnAGuardPath(t *testing.T) {

	connection := model.NewConnection()

	require.NotPanics(t, func() {
		require.NoError(t, NewStripeConnect().Connect(&connection, nil, "localhost"))
	})
}

// TestStripeConnect_LifecycleIsOtherwiseANoOp confirms BeforeSave, Refresh, and Disconnect
// write nothing. Disconnect notably leaves the registered webhook in place at Stripe.
func TestStripeConnect_LifecycleIsOtherwiseANoOp(t *testing.T) {

	connection := newTestConnection()
	connection.Data.SetString("webhook", "we_1PabcdEFGH")
	before := connection

	require.NoError(t, NewStripeConnect().BeforeSave(&connection, mapof.NewString()))
	require.NoError(t, NewStripeConnect().Refresh(&connection, mapof.NewString()))
	require.NoError(t, NewStripeConnect().Disconnect(&connection, mapof.NewString()))

	require.Equal(t, before, connection)
	require.Equal(t, "we_1PabcdEFGH", connection.Data.GetString("webhook"))
}

/******************************************
 * Settings Form
 ******************************************/

// TestStripeConnect_ManualConfigStoresKeysInTheVault confirms both Stripe keys are vault
// paths, and that the client ID, which is not a secret, is not.
func TestStripeConnect_ManualConfigStoresKeysInTheVault(t *testing.T) {

	config := NewStripeConnect().ManualConfig()

	for _, path := range []string{"vault.publishableKey", "vault.restrictedKey"} {
		_, exists := config.Schema.GetElement(path)
		assert.True(t, exists, "missing %s", path)
	}

	_, exists := config.Schema.GetElement("data.clientId")
	assert.True(t, exists, "the client ID is public and belongs in data")
}

// TestStripeConnect_KeyPatternsAcceptRealAndMaskedKeys confirms each key's schema pattern
// admits a live key, a test key, and the asterisk mask the Vault returns on read.
func TestStripeConnect_KeyPatternsAcceptRealAndMaskedKeys(t *testing.T) {

	config := NewStripeConnect().ManualConfig()

	testCases := map[string][]string{
		"vault.publishableKey": {"pk_test_51Abc123", "pk_live_51Abc123", "********"},
		"vault.restrictedKey":  {"rk_test_51Abc123", "rk_live_51Abc123", "********"},
	}

	for path, values := range testCases {

		t.Run(path, func(t *testing.T) {

			element, exists := config.Schema.GetElement(path)
			require.True(t, exists)

			stringElement, isString := element.(schema.String)
			require.True(t, isString)

			pattern, err := regexp.Compile(stringElement.Pattern)
			require.NoError(t, err)

			for _, value := range values {
				assert.True(t, pattern.MatchString(value), "pattern must admit %q", value)
			}
		})
	}
}

// TestStripeConnect_LiveModeEnumDisagreesWithItsLabels pins a mismatch in the settings form:
// the schema admits only "false" and "true" while the select offers SANDBOX and LIVE.
func TestStripeConnect_LiveModeEnumDisagreesWithItsLabels(t *testing.T) {

	config := NewStripeConnect().ManualConfig()

	element, exists := config.Schema.GetElement("data.liveMode")
	require.True(t, exists)

	stringElement, isString := element.(schema.String)
	require.True(t, isString)
	require.Equal(t, []string{"false", "true"}, stringElement.Enum)

	offered := selectOptionValues(config, "data.liveMode")
	require.Equal(t, []string{"SANDBOX", "LIVE"}, offered)

	for _, value := range offered {
		assert.NotContains(t, stringElement.Enum, value, "neither offered value is in the schema enum")
	}
}

// TestStripeConnect_LiveModeCannotBeSaved pins the consequence of that mismatch. Both values
// the select offers are rejected by the schema and dropped without an error, so liveMode stays
// empty and the handler's `liveMode == "LIVE"` test can never be true.
func TestStripeConnect_LiveModeCannotBeSaved(t *testing.T) {

	config := NewStripeConnect().ManualConfig()

	testCases := map[string]string{
		"LIVE":    "",
		"SANDBOX": "",
		"true":    "true",
		"false":   "false",
	}

	for posted, expectedStored := range testCases {

		t.Run("posted="+posted, func(t *testing.T) {

			value := newTestFormValue()

			// A rejected value is logged, not returned, so the admin sees a successful save
			require.NoError(t, config.SetURLValues(&value, url.Values{
				"data.clientId":        []string{"ca_1Abc123"},
				"vault.publishableKey": []string{"pk_test_51Abc123"},
				"vault.restrictedKey":  []string{"rk_test_51Abc123"},
				"data.liveMode":        []string{posted},
				"active":               []string{"true"},
			}, testLookupProvider{}))

			stored := value.GetMap("data").GetString("liveMode")
			require.Equal(t, expectedStored, stored)
			require.NotEqual(t, "LIVE", stored, "handler.stripeConnect reads liveMode == LIVE")
		})
	}
}

// TestStripeConnect_PayPalLiveModeIsDeclaredCorrectly contrasts the two payment forms:
// PayPal declares the same values its select offers, so its live mode does save.
func TestStripeConnect_PayPalLiveModeIsDeclaredCorrectly(t *testing.T) {

	config := NewPayPal().ManualConfig()

	element, exists := config.Schema.GetElement("data.liveMode")
	require.True(t, exists)

	stringElement, isString := element.(schema.String)
	require.True(t, isString)

	for _, offered := range selectOptionValues(config, "data.liveMode") {
		assert.Contains(t, stringElement.Enum, offered)
	}

	value := newTestFormValue()
	require.NoError(t, config.SetURLValues(&value, url.Values{
		"data.bnCode":   []string{"EMISSARY_SP_PPCP"},
		"data.liveMode": []string{"LIVE"},
	}, testLookupProvider{}))

	require.Equal(t, "LIVE", value.GetMap("data").GetString("liveMode"))
}
