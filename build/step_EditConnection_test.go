package build

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service/providers"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

// brokenProvider is a ManualProvider whose form writes a path its schema never declares,
// which is the mistake that shipped twice in service/providers before this gate existed.
type brokenProvider struct {
	providers.Null
}

// ManualConfig returns a form with one element that the schema does not cover
func (brokenProvider) ManualConfig() form.Form {
	return form.Form{
		Schema: schema.Schema{
			Element: schema.Object{
				Properties: schema.ElementMap{
					"data": schema.Object{
						Properties: schema.ElementMap{
							"declared": schema.String{},
						},
					},
				},
			},
		},
		Element: form.Element{
			Type: "layout-vertical",
			Children: []form.Element{
				{Type: "text", Path: "data.declared"},
				{Type: "text", Path: "data.undeclared"},
			},
		},
	}
}

// brokenShowIfProvider is a ManualProvider whose show-if names a field the schema omits,
// which silently renders the field as always hidden.
type brokenShowIfProvider struct {
	providers.Null
}

// ManualConfig returns a form whose show-if references an undeclared field
func (brokenShowIfProvider) ManualConfig() form.Form {
	return form.Form{
		Schema: schema.Schema{
			Element: schema.Object{
				Properties: schema.ElementMap{
					"data": schema.Object{
						Properties: schema.ElementMap{
							"declared": schema.String{},
						},
					},
				},
			},
		},
		Element: form.Element{
			Type: "layout-vertical",
			Children: []form.Element{
				{
					Type:    "text",
					Path:    "data.declared",
					Options: mapof.Any{"show-if": "data.missing == YES"},
				},
			},
		},
	}
}

// TestProviderSettingsForm_AcceptsEveryRealProvider confirms the gate passes all eleven
// ManualProviders, so adding it breaks no existing connection settings page.
func TestProviderSettingsForm_AcceptsEveryRealProvider(t *testing.T) {

	allProviders := map[string]providers.Provider{
		"Bluesky":             providers.NewBluesky(),
		"GeocodeAddress":      providers.NewGeocodeAddress(),
		"GeocodeAutocomplete": providers.NewGeocodeAutocomplete(),
		"GeocodeNetwork":      providers.NewGeocodeNetwork(),
		"GeocodeTiles":        providers.NewGeocodeTiles(),
		"GeocodeTimezone":     providers.NewGeocodeTimezone(),
		"Giphy":               providers.NewGiphy(),
		"PayPal":              providers.NewPayPal(),
		"Stripe":              providers.NewStripe(),
		"StripeConnect":       providers.NewStripeConnect(),
		"Unsplash":            providers.NewUnsplash(),
	}

	for name, provider := range allProviders {

		t.Run(name, func(t *testing.T) {

			result, err := providerSettingsForm(provider)

			require.NoError(t, err)
			require.NotEmpty(t, result.Element.Children)
		})
	}
}

// TestProviderSettingsForm_RejectsAnUndeclaredPath confirms the gate refuses a form that would
// drop a field on save, rather than rendering it and eating the value.
func TestProviderSettingsForm_RejectsAnUndeclaredPath(t *testing.T) {

	_, err := providerSettingsForm(brokenProvider{})
	require.Error(t, err)
}

// TestProviderSettingsForm_RejectsAnUndeclaredShowIfField confirms the gate also covers the
// other half of form.Validate, which a settings page would otherwise render as always hidden.
func TestProviderSettingsForm_RejectsAnUndeclaredShowIfField(t *testing.T) {

	_, err := providerSettingsForm(brokenShowIfProvider{})
	require.Error(t, err)
}

// TestProviderSettingsForm_RejectsANonManualProvider confirms an OAuth-only Provider is turned
// away here rather than at the render call
func TestProviderSettingsForm_RejectsANonManualProvider(t *testing.T) {

	_, err := providerSettingsForm(providers.Null{})
	require.Error(t, err)
}

// TestProviderSettingsForm_ReturnsAZeroFormOnError confirms the caller never receives a
// half-built form alongside an error
func TestProviderSettingsForm_ReturnsAZeroFormOnError(t *testing.T) {

	result, err := providerSettingsForm(brokenProvider{})

	require.Error(t, err)
	require.Equal(t, form.Form{}, result)
}

// TestProviderSettingsForm_StripeConnectLiveModeIsReachable drives the fixed payments defect
// through this gate, because a Domain that cannot select LIVE processes against test keys.
func TestProviderSettingsForm_StripeConnectLiveModeIsReachable(t *testing.T) {

	result, err := providerSettingsForm(providers.NewStripeConnect())
	require.NoError(t, err)

	element, exists := result.Schema.GetElement("data.liveMode")
	require.True(t, exists)

	stringElement, isString := element.(schema.String)
	require.True(t, isString)
	require.Contains(t, stringElement.Enum, "LIVE")

	connection := model.NewConnection()
	require.NoError(t, result.Schema.Set(&connection, "data.liveMode", "LIVE"))
	require.Equal(t, "LIVE", connection.Data.GetString("liveMode"))
}
