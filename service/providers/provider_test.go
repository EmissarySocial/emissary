package providers

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"github.com/benpate/form/widget"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// TestMain registers the built-in form widgets once, because every ManualConfig
// renders through the form package's global widget registry.
func TestMain(m *testing.M) {
	widget.UseAll()
	os.Exit(m.Run())
}

/******************************************
 * Compile-Time Interface Assertions
 ******************************************/

var _ Provider = Null{}
var _ Provider = Bluesky{}
var _ Provider = GeocodeAddress{}
var _ Provider = GeocodeAutocomplete{}
var _ Provider = GeocodeNetwork{}
var _ Provider = GeocodeTiles{}
var _ Provider = GeocodeTimezone{}
var _ Provider = Giphy{}
var _ Provider = PayPal{}
var _ Provider = Stripe{}
var _ Provider = StripeConnect{}
var _ Provider = Unsplash{}

var _ ManualProvider = Bluesky{}
var _ ManualProvider = GeocodeAddress{}
var _ ManualProvider = GeocodeAutocomplete{}
var _ ManualProvider = GeocodeNetwork{}
var _ ManualProvider = GeocodeTiles{}
var _ ManualProvider = GeocodeTimezone{}
var _ ManualProvider = Giphy{}
var _ ManualProvider = PayPal{}
var _ ManualProvider = Stripe{}
var _ ManualProvider = StripeConnect{}
var _ ManualProvider = Unsplash{}

/******************************************
 * Shared Test Helpers
 ******************************************/

// namedProvider pairs a Provider with the name used in subtest output
type namedProvider struct {
	Name     string
	Provider Provider
}

// allProviders returns every Provider in this package, Null included
func allProviders() []namedProvider {
	return []namedProvider{
		{"Null", Null{}},
		{"Bluesky", NewBluesky()},
		{"GeocodeAddress", NewGeocodeAddress()},
		{"GeocodeAutocomplete", NewGeocodeAutocomplete()},
		{"GeocodeNetwork", NewGeocodeNetwork()},
		{"GeocodeTiles", NewGeocodeTiles()},
		{"GeocodeTimezone", NewGeocodeTimezone()},
		{"Giphy", NewGiphy()},
		{"PayPal", NewPayPal()},
		{"Stripe", NewStripe()},
		{"StripeConnect", NewStripeConnect()},
		{"Unsplash", NewUnsplash()},
	}
}

// allManualConfigs returns the ManualConfig form of every ManualProvider, keyed by name
func allManualConfigs(t *testing.T) map[string]form.Form {

	t.Helper()

	result := make(map[string]form.Form)

	for _, named := range allProviders() {

		manual, isManual := named.Provider.(ManualProvider)

		if !isManual {
			continue
		}

		result[named.Name] = manual.ManualConfig()
	}

	return result
}

// newTestConnection returns a Connection with every map initialized, as
// model.NewConnection does, plus a token that is still valid.
func newTestConnection() model.Connection {

	connection := model.NewConnection()
	connection.Token = &oauth2.Token{AccessToken: "test-access-token", Expiry: time.Now().Add(time.Hour)}

	return connection
}

// newTestFormValue returns the object shape that a Connection settings form edits
func newTestFormValue() mapof.Any {
	return mapof.Any{
		"type":   "",
		"active": false,
		"data":   mapof.NewAny(),
		"vault":  mapof.NewString(),
	}
}

// testLookupProvider stands in for the Domain's real LookupProvider
type testLookupProvider struct{}

// Group returns a fixed set of LookupCodes for the named group
func (testLookupProvider) Group(name string) form.LookupGroup {

	switch name {

	// Tile groups are the values that the "geocode-tiles" select-group writes into data.provider
	case "geocode-tiles":
		return testLookupGroup{[]form.LookupCode{
			{Group: "Open Street Map", Value: "OPEN-STREET-MAPS-STANDARD", Label: "Standard"},
			{Group: "Maptiler", Value: "MAPTILER-STREETS", Label: "Streets"},
			{Group: "Custom", Value: "CUSTOM", Label: "Custom", Href: "{href}"},
		}}

	case "groups":
		return testLookupGroup{[]form.LookupCode{
			{Value: "5f8e1d2c3b4a5968770a1b2c", Label: "Members"},
			{Value: "5f8e1d2c3b4a5968770a1b2d", Label: "Staff"},
		}}
	}

	return testLookupGroup{}
}

// testLookupGroup is a read-only LookupGroup backed by a fixed slice
type testLookupGroup struct {
	codes []form.LookupCode
}

// Get returns this group's LookupCodes
func (group testLookupGroup) Get() []form.LookupCode {
	return group.codes
}

/******************************************
 * Interface Conformance
 ******************************************/

// TestProvider_NullIsNotAManualProvider confirms the do-nothing Provider offers no settings form
func TestProvider_NullIsNotAManualProvider(t *testing.T) {

	var provider any = Null{}

	_, isManual := provider.(ManualProvider)
	require.False(t, isManual, "Null has nothing to configure")
}

// TestProvider_NoProviderImplementsOAuthProvider pins the fact that OAuthProvider has
// no implementations, which makes service.Domain's three OAuthConfig call sites unreachable.
func TestProvider_NoProviderImplementsOAuthProvider(t *testing.T) {

	for _, named := range allProviders() {

		t.Run(named.Name, func(t *testing.T) {
			var provider any = named.Provider
			_, isOAuth := provider.(OAuthProvider)
			assert.False(t, isOAuth, "no provider implements OAuthProvider today")
		})
	}
}

/******************************************
 * Lifecycle Contract
 ******************************************/

// TestProvider_LifecycleSucceedsOnLocalhost drives all four lifecycle methods of every
// Provider. A localhost host and a valid Token keep PayPal and StripeConnect on their
// guard paths, so no method reaches the network.
func TestProvider_LifecycleSucceedsOnLocalhost(t *testing.T) {

	for _, named := range allProviders() {

		t.Run(named.Name, func(t *testing.T) {

			connection := newTestConnection()
			vault := mapof.NewString()

			assert.NoError(t, named.Provider.BeforeSave(&connection, vault))
			assert.NoError(t, named.Provider.Connect(&connection, vault, "localhost"))
			assert.NoError(t, named.Provider.Refresh(&connection, vault))
			assert.NoError(t, named.Provider.Disconnect(&connection, vault))
		})
	}
}

// TestProvider_LifecycleAcceptsNilVault confirms no Provider dereferences the vault map.
// Reading a nil map is legal in Go, so a Provider that only reads it stays nil-safe.
func TestProvider_LifecycleAcceptsNilVault(t *testing.T) {

	for _, named := range allProviders() {

		t.Run(named.Name, func(t *testing.T) {

			connection := newTestConnection()

			assert.NoError(t, named.Provider.BeforeSave(&connection, nil))
			assert.NoError(t, named.Provider.Connect(&connection, nil, "localhost"))
			assert.NoError(t, named.Provider.Refresh(&connection, nil))
			assert.NoError(t, named.Provider.Disconnect(&connection, nil))
		})
	}
}

// TestProvider_LifecycleAcceptsZeroConnection drives every Provider against a Connection
// whose Data and Vault maps are nil, as a record written before either field existed would be.
func TestProvider_LifecycleAcceptsZeroConnection(t *testing.T) {

	for _, named := range allProviders() {

		t.Run(named.Name, func(t *testing.T) {

			// The Token stays valid so PayPal.Refresh keeps to its guard path
			connection := model.Connection{Token: newTestConnection().Token}
			vault := mapof.NewString()

			assert.NoError(t, named.Provider.BeforeSave(&connection, vault))
			assert.NoError(t, named.Provider.Refresh(&connection, vault))
			assert.NoError(t, named.Provider.Disconnect(&connection, vault))
		})
	}
}

// TestProvider_OnlyBlueskyChangesTheConnection confirms BeforeSave is a no-op everywhere
// but Bluesky, which derives Active from the allowType setting.
func TestProvider_OnlyBlueskyChangesTheConnection(t *testing.T) {

	for _, named := range allProviders() {

		t.Run(named.Name, func(t *testing.T) {

			connection := newTestConnection()
			connection.Data.SetString("sentinel", "untouched")

			require.NoError(t, named.Provider.BeforeSave(&connection, mapof.NewString()))

			assert.Equal(t, "untouched", connection.Data.GetString("sentinel"))

			if named.Name == "Bluesky" {
				assert.True(t, connection.Active, "Bluesky activates a connection with no allowType")
				return
			}

			assert.False(t, connection.Active, "BeforeSave leaves Active alone")
		})
	}
}

/******************************************
 * ManualConfig Contract
 ******************************************/

// TestProvider_ManualConfigValidatesAgainstOwnSchema runs form.Validate over every
// settings form, which rejects a form element or show-if field that the schema omits.
func TestProvider_ManualConfigValidatesAgainstOwnSchema(t *testing.T) {

	// GeocodeTiles declares a "data.href" element that its schema never defines
	const knownBrokenConfig = "GeocodeTiles"

	for name, config := range allManualConfigs(t) {

		t.Run(name, func(t *testing.T) {

			err := config.Validate()

			if name == knownBrokenConfig {
				require.Error(t, err, "the data.href defect is still present -- see geocode-tiles_test.go")
				return
			}

			require.NoError(t, err)
		})
	}
}

// TestProvider_ManualConfigRendersBothViews confirms every settings form draws an editor
// and a viewer without error, against an empty Connection.
func TestProvider_ManualConfigRendersBothViews(t *testing.T) {

	for name, config := range allManualConfigs(t) {

		t.Run(name, func(t *testing.T) {

			editor, err := config.Editor(newTestFormValue(), testLookupProvider{})
			require.NoError(t, err)
			assert.NotEmpty(t, editor)

			viewer, err := config.Viewer(newTestFormValue(), testLookupProvider{})
			require.NoError(t, err)
			assert.NotEmpty(t, viewer)
		})
	}
}

// TestProvider_ManualConfigRendersWithNoLookupProvider confirms a nil LookupProvider does
// not panic, which is how a form renders before the Domain's lookups are wired up.
func TestProvider_ManualConfigRendersWithNoLookupProvider(t *testing.T) {

	for name, config := range allManualConfigs(t) {

		t.Run(name, func(t *testing.T) {
			require.NotPanics(t, func() {
				_, _ = config.Editor(newTestFormValue(), nil)
				_, _ = config.Viewer(newTestFormValue(), nil)
			})
		})
	}
}

// TestProvider_ManualConfigEscapesStoredValues confirms a stored value is HTML-escaped in
// both views. Labels and descriptions are author-written constants and stay unescaped.
func TestProvider_ManualConfigEscapesStoredValues(t *testing.T) {

	const payload = `"><script>alert(1)</script>`

	for name, config := range allManualConfigs(t) {

		t.Run(name, func(t *testing.T) {

			value := newTestFormValue()

			for _, element := range config.Element.AllElements() {
				_ = config.Schema.Set(&value, element.Path, payload)
			}

			editor, err := config.Editor(value, testLookupProvider{})
			require.NoError(t, err)
			assert.NotContains(t, editor, "<script>", "editor escapes stored values")

			viewer, err := config.Viewer(value, testLookupProvider{})
			require.NoError(t, err)
			assert.NotContains(t, viewer, "<script>", "viewer escapes stored values")
		})
	}
}

// TestProvider_HiddenTypeMatchesSchemaEnum confirms the "type" value a form writes is one
// the same form's schema admits. The Type and Provider constant pairs are not all equal,
// so a form can name one where it means the other.
func TestProvider_HiddenTypeMatchesSchemaEnum(t *testing.T) {

	for name, config := range allManualConfigs(t) {

		t.Run(name, func(t *testing.T) {

			written := hiddenTypeValue(config)

			if written == "" {
				t.Skip("this form does not write a type")
			}

			element, exists := config.Schema.GetElement("type")
			require.True(t, exists, "a form that writes type needs it in the schema")

			stringElement, isString := element.(schema.String)
			require.True(t, isString)
			require.Contains(t, stringElement.Enum, written)
		})
	}
}

// TestProvider_ManualConfigPathsAreUnique confirms no settings form writes one path twice,
// which would make the second element silently overwrite the first on save.
func TestProvider_ManualConfigPathsAreUnique(t *testing.T) {

	for name, config := range allManualConfigs(t) {

		t.Run(name, func(t *testing.T) {

			seen := make(map[string]int)

			for _, element := range config.Element.AllElements() {
				seen[element.Path]++
			}

			for path, count := range seen {
				assert.Equal(t, 1, count, "path %q appears %d times", path, count)
			}
		})
	}
}

// TestProvider_ManualConfigLabelsAreSet confirms every form names itself, because the
// admin connection list shows nothing but the label.
func TestProvider_ManualConfigLabelsAreSet(t *testing.T) {

	for name, config := range allManualConfigs(t) {

		t.Run(name, func(t *testing.T) {
			assert.NotEmpty(t, config.Element.Label)
			assert.Equal(t, "layout-vertical", config.Element.Type)
		})
	}
}

/******************************************
 * Internal Test Helpers
 ******************************************/

// hiddenTypeValue returns the value that a form's hidden "type" element writes,
// or an empty string when the form has no such element.
func hiddenTypeValue(config form.Form) string {

	for _, element := range config.Element.AllElements() {

		if element.Path != "type" {
			continue
		}

		if strings.EqualFold(element.Type, "hidden") {
			return element.Options.GetString("value")
		}
	}

	return ""
}

// selectOptionValues returns the LookupCode values that a select element offers
// for the named path, or nil when the path has no enum option.
func selectOptionValues(config form.Form, path string) []string {

	for _, element := range config.Element.AllElements() {

		if element.Path != path {
			continue
		}

		var result []string

		switch enum := element.Options["enum"].(type) {

		case []form.LookupCode:
			for _, lookupCode := range enum {
				result = append(result, lookupCode.Value)
			}

		case []any:
			for _, item := range enum {
				result = append(result, form.ParseLookupCode(item).Value)
			}
		}

		return result
	}

	return nil
}

// indexFormPaths returns the set of paths that a form's elements write
func indexFormPaths(config form.Form) map[string]bool {

	result := make(map[string]bool)

	for _, element := range config.Element.AllElements() {
		result[element.Path] = true
	}

	return result
}
