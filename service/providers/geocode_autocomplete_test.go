package providers

import (
	"net/url"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGeocodeAutocomplete_OfferedProviders pins the service list. Nominatim is commented out
// in the form, so it must not appear here.
func TestGeocodeAutocomplete_OfferedProviders(t *testing.T) {

	config := NewGeocodeAutocomplete().ManualConfig()
	offered := selectOptionValues(config, "data.provider")

	require.Equal(t, []string{"GEOAPIFY", "HERE", "GOOGLE-MAPS", "MAPTILER"}, offered)
	require.NotContains(t, offered, "NOMINATIM")
}

// TestGeocodeAutocomplete_OffersFewerProvidersThanAddressGeocoding pins that the two address
// forms do not offer the same services, so a Domain can autocomplete what it cannot geocode.
func TestGeocodeAutocomplete_OffersFewerProvidersThanAddressGeocoding(t *testing.T) {

	autocomplete := selectOptionValues(NewGeocodeAutocomplete().ManualConfig(), "data.provider")
	address := selectOptionValues(NewGeocodeAddress().ManualConfig(), "data.provider")

	require.Subset(t, address, autocomplete, "every autocomplete service also geocodes")
	require.NotSubset(t, autocomplete, address, "the address form offers more")
}

// TestGeocodeAutocomplete_TypeMatchesTheConnectionLoader confirms the written type value
func TestGeocodeAutocomplete_TypeMatchesTheConnectionLoader(t *testing.T) {

	config := NewGeocodeAutocomplete().ManualConfig()
	require.Equal(t, model.ConnectionTypeGeocodeAutocomplete, hiddenTypeValue(config))
}

// TestGeocodeAutocomplete_CredentialGating walks the two show-if rules together: apiID is for
// Here alone, and apiKey needs any provider at all.
func TestGeocodeAutocomplete_CredentialGating(t *testing.T) {

	testCases := map[string]struct {
		provider       string
		expectedAPIID  string
		expectedAPIKey string
	}{
		"Here takes both":        {"HERE", "app-id", "api-key"},
		"Geoapify takes a key":   {"GEOAPIFY", "", "api-key"},
		"Maptiler takes a key":   {"MAPTILER", "", "api-key"},
		"no provider takes none": {"", "", ""},
	}

	for name, testCase := range testCases {

		t.Run(name, func(t *testing.T) {

			config := NewGeocodeAutocomplete().ManualConfig()
			value := newTestFormValue()

			require.NoError(t, config.SetURLValues(&value, url.Values{
				"data.provider": []string{testCase.provider},
				"data.apiID":    []string{"app-id"},
				"data.apiKey":   []string{"api-key"},
			}, testLookupProvider{}))

			data := value.GetMap("data")
			assert.Equal(t, testCase.expectedAPIID, data.GetString("apiID"))
			assert.Equal(t, testCase.expectedAPIKey, data.GetString("apiKey"))
		})
	}
}

// TestGeocodeAutocomplete_LifecycleIsANoOp confirms this provider only supplies a form
func TestGeocodeAutocomplete_LifecycleIsANoOp(t *testing.T) {

	connection := newTestConnection()
	before := connection

	require.NoError(t, NewGeocodeAutocomplete().BeforeSave(&connection, mapof.NewString()))
	require.NoError(t, NewGeocodeAutocomplete().Connect(&connection, mapof.NewString(), "example.com"))
	require.NoError(t, NewGeocodeAutocomplete().Refresh(&connection, mapof.NewString()))
	require.NoError(t, NewGeocodeAutocomplete().Disconnect(&connection, mapof.NewString()))

	require.Equal(t, before, connection)
}
