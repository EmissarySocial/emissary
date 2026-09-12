package providers

import (
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/uri"
	"github.com/stretchr/testify/require"
)

// A Connection settings form is the one place this package takes outside input: an admin posts
// arbitrary url.Values, and whatever survives is rendered straight back. These targets fuzz
// both directions. None of them reaches the network.

// orderedManualConfigs returns every ManualConfig in a fixed order, so a fuzz corpus entry
// keeps naming the same provider across runs.
func orderedManualConfigs() []form.Form {

	result := make([]form.Form, 0)

	for _, named := range allProviders() {

		if manual, isManual := named.Provider.(ManualProvider); isManual {
			result = append(result, manual.ManualConfig())
		}
	}

	return result
}

// FuzzManualConfigSetURLValues posts one arbitrary field at one arbitrary provider's form.
// Properties: the post never panics, it never returns an error, and no key outside the form's
// own schema reaches the saved object.
func FuzzManualConfigSetURLValues(f *testing.F) {

	f.Add(0, "data.provider", "GEOAPIFY")
	f.Add(1, "data.apiKey", "secret")
	f.Add(2, "data.href", "https://tile.example.com/{z}/{x}/{y}.png")
	f.Add(3, "data.allowType", "NONE")
	f.Add(4, "active", "true")
	f.Add(5, "type", model.ConnectionTypeUserPayment)
	f.Add(6, "vault.restrictedKey", "rk_live_1")
	f.Add(7, "data.shareGroups", "5f8e1d2c3b4a5968770a1b2c")
	f.Add(0, "", "")
	f.Add(0, "data.provider", "\x00\xff")
	f.Add(0, "..", "..")
	f.Add(0, "data.provider.deeper.still", "x")
	f.Add(0, strings.Repeat("a", 1024), strings.Repeat("b", 4096))

	f.Fuzz(func(t *testing.T, index int, key string, postedValue string) {

		configs := orderedManualConfigs()
		require.NotEmpty(t, configs)

		config := configs[positiveModulo(index, len(configs))]

		value := newTestFormValue()

		require.NoError(t, config.SetURLValues(&value, map[string][]string{
			key: {postedValue},
		}, testLookupProvider{}))

		requireOnlySchemaPaths(t, config, value, "data")
		requireOnlySchemaPaths(t, config, value, "vault")
	})
}

// FuzzManualConfigEditor stores one arbitrary value at every path a form writes, then renders
// both views. Properties: no stored string keeps an angle bracket, because every schema here
// sanitizes on write, and rendering the result never panics or errors.
func FuzzManualConfigEditor(f *testing.F) {

	f.Add(0, `"><script>alert(1)</script>`)
	f.Add(1, "<img src=x onerror=alert(1)>")
	f.Add(2, "javascript:alert(1)")
	f.Add(3, "")
	f.Add(4, "\x00")
	f.Add(5, "\xff\xfe invalid utf8")
	f.Add(6, "'single' \"double\" `backtick`")
	f.Add(7, strings.Repeat("<", 4096))
	f.Add(8, "0<")
	f.Add(9, "a<<b>>c")

	f.Fuzz(func(t *testing.T, index int, storedValue string) {

		configs := orderedManualConfigs()
		require.NotEmpty(t, configs)

		config := configs[positiveModulo(index, len(configs))]
		value := newTestFormValue()

		for _, element := range config.Element.AllElements() {

			// A value the schema refuses leaves the old one in place, which is also fine
			_ = config.Schema.Set(&value, element.Path, storedValue)

			stored, err := config.Schema.Get(value, element.Path)

			if err != nil {
				continue
			}

			text, isString := stored.(string)

			if !isString {
				continue
			}

			require.NotContains(t, text, "<", "a stored value kept an angle bracket")
			require.NotContains(t, text, ">", "a stored value kept an angle bracket")
		}

		_, err := config.Editor(value, testLookupProvider{})
		require.NoError(t, err)

		_, err = config.Viewer(value, testLookupProvider{})
		require.NoError(t, err)
	})
}

// FuzzBlueskyBeforeSave drives the one lifecycle hook in this package that makes a decision.
// Properties: it never panics or errors, only the literal "NONE" disables the bridge, and a
// second call agrees with the first.
func FuzzBlueskyBeforeSave(f *testing.F) {

	f.Add("NONE")
	f.Add("GROUPS")
	f.Add("ALL")
	f.Add("")
	f.Add("none")
	f.Add(" NONE ")
	f.Add("NONE\x00")
	f.Add("\xff\xfe")
	f.Add(strings.Repeat("NONE", 1024))

	f.Fuzz(func(t *testing.T, allowType string) {

		connection := model.NewConnection()
		connection.Data.SetString("allowType", allowType)

		require.NoError(t, NewBluesky().BeforeSave(&connection, mapof.NewString()))
		require.Equal(t, allowType != "NONE", connection.Active)

		first := connection.Active

		require.NoError(t, NewBluesky().BeforeSave(&connection, mapof.NewString()))
		require.Equal(t, first, connection.Active, "BeforeSave is idempotent")
	})
}

// FuzzStripeConnectConnectLocalHost fuzzes the host that gates webhook registration. Only hosts
// that uri reports as local are passed to Connect, because a public host would reach the Stripe
// API. Properties: the guarded call never panics or errors, and registers no webhook.
func FuzzStripeConnectConnectLocalHost(f *testing.F) {

	f.Add("localhost")
	f.Add("127.0.0.1:8080")
	f.Add("[::1]")
	f.Add("10.0.0.1")
	f.Add("emissary.local")
	f.Add("host.docker.internal")
	f.Add("")
	f.Add("..")
	f.Add("\x00")
	f.Add("http://user:pass@127.0.0.1:99999/path?q=1#frag")
	f.Add(strings.Repeat("a.", 2048) + "local")

	f.Fuzz(func(t *testing.T, host string) {

		// RULE: Never hand Connect a host that would let it reach api.stripe.com
		if !uri.IsLocalHostname(host) {
			t.Skip("a public host would make a live request")
		}

		connection := model.NewConnection()

		require.NoError(t, NewStripeConnect().Connect(&connection, mapof.NewString(), host))
		require.Empty(t, connection.Data.GetString("webhook"))
		require.False(t, connection.Vault.HasString("webhookSecret"))
	})
}

/******************************************
 * Fuzz Helpers
 ******************************************/

// positiveModulo maps any int, negative values included, onto [0, length)
func positiveModulo(value int, length int) int {

	result := value % length

	if result < 0 {
		result += length
	}

	return result
}

// requireOnlySchemaPaths confirms every key in the named sub-map is declared in the form's schema
func requireOnlySchemaPaths(t *testing.T, config form.Form, value mapof.Any, prefix string) {

	t.Helper()

	for key := range value.GetMap(prefix) {
		_, exists := config.Schema.GetElement(prefix + "." + key)
		require.True(t, exists, "%s.%s is not declared in the schema", prefix, key)
	}
}
