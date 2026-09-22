package providers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"sync"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// StripeConnect.Connect registers a webhook with the Stripe API unless one of its two guards
// fires first. The guards are exercised here, and the Stripe calls against a fake Stripe API below.

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

// TestStripeConnect_ConnectSkipsAWebhookThatHasItsSecret confirms a Connection that already
// carries a webhook ID and its secret is left alone, so a re-save calls Stripe for nothing.
func TestStripeConnect_ConnectSkipsAWebhookThatHasItsSecret(t *testing.T) {

	stripe, requests := newFakeStripe(t, http.StatusOK, `{"id":"we_new","secret":"whsec_new"}`, http.StatusOK)

	connection := model.NewConnection()
	connection.Data.SetString("webhook", "we_1PabcdEFGH")
	vault := mapof.String{"restrictedKey": "rk_test_1", "webhookSecret": "whsec_old"}

	require.NoError(t, stripe.Connect(&connection, vault, "example.com"))
	require.Equal(t, "we_1PabcdEFGH", connection.Data.GetString("webhook"), "the existing webhook is preserved")
	require.Empty(t, *requests, "nothing is sent to Stripe")
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

// TestStripeConnect_LiveModeEnumMatchesItsLabels confirms the schema admits exactly the two
// values the select offers. It did not until 2026-09-11, which made live mode unreachable.
func TestStripeConnect_LiveModeEnumMatchesItsLabels(t *testing.T) {

	config := NewStripeConnect().ManualConfig()

	element, exists := config.Schema.GetElement("data.liveMode")
	require.True(t, exists)

	stringElement, isString := element.(schema.String)
	require.True(t, isString)

	offered := selectOptionValues(config, "data.liveMode")
	require.Equal(t, []string{"SANDBOX", "LIVE"}, offered)
	require.Equal(t, offered, stringElement.Enum)
}

// TestStripeConnect_LiveModeSaves confirms the value the select offers survives a save, and
// that no other spelling does. Both readers in handler.stripeConnect compare against "LIVE".
func TestStripeConnect_LiveModeSaves(t *testing.T) {

	config := NewStripeConnect().ManualConfig()

	testCases := map[string]string{
		"LIVE":    "LIVE",
		"SANDBOX": "SANDBOX",
		"true":    "",
		"false":   "",
		"live":    "",
	}

	for posted, expectedStored := range testCases {

		t.Run("posted="+posted, func(t *testing.T) {

			value := newTestFormValue()

			require.NoError(t, config.SetURLValues(&value, url.Values{
				"data.clientId":        []string{"ca_1Abc123"},
				"vault.publishableKey": []string{"pk_test_51Abc123"},
				"vault.restrictedKey":  []string{"rk_test_51Abc123"},
				"data.liveMode":        []string{posted},
				"active":               []string{"true"},
			}, testLookupProvider{}))

			require.Equal(t, expectedStored, value.GetMap("data").GetString("liveMode"))
		})
	}
}

// TestStripeConnect_LiveModeReachesTheHandlersTest closes the loop on the defect by asserting
// the comparison that handler.stripeConnect makes, rather than only the stored value.
func TestStripeConnect_LiveModeReachesTheHandlersTest(t *testing.T) {

	config := NewStripeConnect().ManualConfig()
	value := newTestFormValue()

	require.NoError(t, config.SetURLValues(&value, url.Values{
		"data.clientId":        []string{"ca_1Abc123"},
		"vault.publishableKey": []string{"pk_live_51Abc123"},
		"vault.restrictedKey":  []string{"rk_live_51Abc123"},
		"data.liveMode":        []string{"LIVE"},
		"active":               []string{"true"},
	}, testLookupProvider{}))

	liveMode := value.GetMap("data").GetString("liveMode") == "LIVE"
	require.True(t, liveMode, "an admin who selects LIVE gets live mode")
}

/******************************************
 * Stripe API
 *
 * These run Connect against a fake Stripe API.  Stripe returns an
 * endpoint's signing secret only when it creates the endpoint, so a
 * webhook ID stored without its secret can only be fixed by replacing it.
 ******************************************/

// newFakeStripe returns a StripeConnect pointed at a fake Stripe API that answers endpoint creation
// and deletion with the given statuses, plus the "METHOD /path" of every request it received.
func newFakeStripe(t *testing.T, createStatus int, createBody string, deleteStatus int) (StripeConnect, *[]string) {

	t.Helper()

	var lock sync.Mutex
	requests := make([]string, 0)

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {

		lock.Lock()
		requests = append(requests, request.Method+" "+request.URL.Path)
		lock.Unlock()

		response.Header().Set("Content-Type", "application/json")

		if request.Method == http.MethodDelete {
			response.WriteHeader(deleteStatus)
			_, _ = response.Write([]byte(`{"deleted":true}`)) // the test inspects the request, not this body
			return
		}

		response.WriteHeader(createStatus)
		_, _ = response.Write([]byte(createBody)) // the test inspects the request, not this body
	}))

	t.Cleanup(server.Close)

	return StripeConnect{apiBase: server.URL, allowPrivateIPs: true}, &requests
}

// stripeReportRecorder is a derp reporter that counts the errors reported to it
type stripeReportRecorder struct {
	lock   sync.Mutex
	errors []error
}

// Report records one error
func (recorder *stripeReportRecorder) Report(err error) {
	recorder.lock.Lock()
	defer recorder.lock.Unlock()
	recorder.errors = append(recorder.errors, err)
}

// recordStripeReports captures every derp report until the test ends
func recordStripeReports(t *testing.T) *stripeReportRecorder {

	t.Helper()

	recorder := &stripeReportRecorder{}
	derp.SetPlugins(recorder)

	t.Cleanup(func() {
		derp.SetPlugins()
	})

	return recorder
}

// storedWebhookSecret returns the webhook secret that Connect left in the Connection's vault
func storedWebhookSecret(t *testing.T, connection *model.Connection) string {

	t.Helper()

	// Nothing is sealed yet, so Decrypt returns the plaintext without using a key
	values, err := connection.Vault.Decrypt(nil, "webhookSecret")
	require.NoError(t, err)

	return values.GetString("webhookSecret")
}

// TestStripeConnect_ConnectRegistersAWebhook confirms a new Connection registers one endpoint, with
// the restricted key, and keeps both the endpoint ID and its signing secret.
func TestStripeConnect_ConnectRegistersAWebhook(t *testing.T) {

	stripe, requests := newFakeStripe(t, http.StatusOK, `{"id":"we_new","secret":"whsec_new"}`, http.StatusOK)
	connection := model.NewConnection()

	require.NoError(t, stripe.Connect(&connection, mapof.String{"restrictedKey": "rk_test_1"}, "https://example.com"))

	require.Equal(t, []string{"POST /v1/webhook_endpoints"}, *requests)
	require.Equal(t, "we_new", connection.Data.GetString("webhook"))
	require.Equal(t, "whsec_new", storedWebhookSecret(t, &connection))
}

// TestStripeConnect_ConnectReplacesAWebhookWithoutItsSecret confirms a webhook ID stored without its
// secret is replaced: a new endpoint is created first, then the old one is deleted.
func TestStripeConnect_ConnectReplacesAWebhookWithoutItsSecret(t *testing.T) {

	reports := recordStripeReports(t)
	stripe, requests := newFakeStripe(t, http.StatusOK, `{"id":"we_new","secret":"whsec_new"}`, http.StatusOK)

	connection := model.NewConnection()
	connection.Data.SetString("webhook", "we_old")

	require.NoError(t, stripe.Connect(&connection, mapof.String{"restrictedKey": "rk_test_1"}, "https://example.com"))

	require.Equal(t, []string{"POST /v1/webhook_endpoints", "DELETE /v1/webhook_endpoints/we_old"}, *requests)
	require.Equal(t, "we_new", connection.Data.GetString("webhook"))
	require.Equal(t, "whsec_new", storedWebhookSecret(t, &connection))
	require.Empty(t, reports.errors)
}

// TestStripeConnect_ConnectReplacesAWebhookThatIsAlreadyGone confirms Stripe's 404 for an endpoint
// that no longer exists counts as success, and is not reported.
func TestStripeConnect_ConnectReplacesAWebhookThatIsAlreadyGone(t *testing.T) {

	reports := recordStripeReports(t)
	stripe, _ := newFakeStripe(t, http.StatusOK, `{"id":"we_new","secret":"whsec_new"}`, http.StatusNotFound)

	connection := model.NewConnection()
	connection.Data.SetString("webhook", "we_old")

	require.NoError(t, stripe.Connect(&connection, mapof.String{"restrictedKey": "rk_test_1"}, "https://example.com"))

	require.Equal(t, "we_new", connection.Data.GetString("webhook"))
	require.Equal(t, "whsec_new", storedWebhookSecret(t, &connection))
	require.Empty(t, reports.errors, "an endpoint that is already gone is the outcome we wanted")
}

// TestStripeConnect_ConnectKeepsTheReplacementWhenTheDeleteFails confirms a failed delete is reported
// but not returned, because the new endpoint is registered and its secret must still be saved.
func TestStripeConnect_ConnectKeepsTheReplacementWhenTheDeleteFails(t *testing.T) {

	reports := recordStripeReports(t)
	stripe, _ := newFakeStripe(t, http.StatusOK, `{"id":"we_new","secret":"whsec_new"}`, http.StatusInternalServerError)

	connection := model.NewConnection()
	connection.Data.SetString("webhook", "we_old")

	require.NoError(t, stripe.Connect(&connection, mapof.String{"restrictedKey": "rk_test_1"}, "https://example.com"))

	require.Equal(t, "we_new", connection.Data.GetString("webhook"))
	require.Equal(t, "whsec_new", storedWebhookSecret(t, &connection))
	require.Len(t, reports.errors, 1, "the endpoint left behind is reported")
	require.NotContains(t, reports.errors[0].Error(), "rk_test_1", "the report never carries the restricted key")
}

// TestStripeConnect_ConnectReturnsARefusedWebhook confirms a failed create changes nothing, and
// leaves the old endpoint in place.
func TestStripeConnect_ConnectReturnsARefusedWebhook(t *testing.T) {

	stripe, requests := newFakeStripe(t, http.StatusUnauthorized, `{"error":{"message":"Invalid API Key"}}`, http.StatusOK)

	connection := model.NewConnection()
	connection.Data.SetString("webhook", "we_old")

	require.Error(t, stripe.Connect(&connection, mapof.String{"restrictedKey": "rk_test_1"}, "https://example.com"))

	require.Equal(t, []string{"POST /v1/webhook_endpoints"}, *requests, "the old endpoint is not deleted")
	require.Equal(t, "we_old", connection.Data.GetString("webhook"))
	require.Empty(t, storedWebhookSecret(t, &connection))
}

// TestStripeConnect_ConnectRefusesAWebhookWithNoSecret confirms an endpoint created without a secret
// is never stored, because it could verify nothing and would never be replaced.
func TestStripeConnect_ConnectRefusesAWebhookWithNoSecret(t *testing.T) {

	stripe, requests := newFakeStripe(t, http.StatusOK, `{"id":"we_new"}`, http.StatusOK)

	connection := model.NewConnection()
	connection.Data.SetString("webhook", "we_old")

	require.Error(t, stripe.Connect(&connection, mapof.String{"restrictedKey": "rk_test_1"}, "https://example.com"))

	require.Equal(t, []string{"POST /v1/webhook_endpoints"}, *requests, "the old endpoint is not deleted")
	require.Equal(t, "we_old", connection.Data.GetString("webhook"))
	require.Empty(t, storedWebhookSecret(t, &connection))
}

// TestStripeConnect_Endpoint confirms every constructor calls the real Stripe API, and that only the
// field tests set moves it.
func TestStripeConnect_Endpoint(t *testing.T) {

	require.Equal(t, "https://api.stripe.com/v1/webhook_endpoints", NewStripeConnect().endpoint("/v1/webhook_endpoints"))
	require.Equal(t, "https://api.stripe.com/v1/webhook_endpoints", StripeConnect{}.endpoint("/v1/webhook_endpoints"))
	require.Equal(t, "http://127.0.0.1:9/v1/webhook_endpoints", StripeConnect{apiBase: "http://127.0.0.1:9"}.endpoint("/v1/webhook_endpoints"))
}
