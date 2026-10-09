package providers

import (
	"net/url"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGeocodeAddress_OfferedProviders pins the service list, because service.GeocodeAddress
// switches on these exact values and falls through to no geocoder at all on a mismatch.
func TestGeocodeAddress_OfferedProviders(t *testing.T) {

	config := NewGeocodeAddress().ManualConfig()

	require.Equal(t, []string{
		"GEOAPIFY",
		"HERE",
		"GOOGLE-MAPS",
		"MAPTILER",
		"OPEN-STREET-MAP",
		"GEOCODIO",
	}, selectOptionValues(config, "data.provider"))
}

// TestGeocodeAddress_TypeMatchesTheConnectionLoader confirms the hidden type element writes
// the value that service.Connection.LoadActiveByType queries for.
func TestGeocodeAddress_TypeMatchesTheConnectionLoader(t *testing.T) {

	config := NewGeocodeAddress().ManualConfig()
	require.Equal(t, model.ConnectionTypeGeocodeAddress, hiddenTypeValue(config))
}

// TestGeocodeAddress_APIIDOnlyShownForHere confirms the Here-specific field is the only one
// gated on a single provider, and that a posted value is dropped for every other one.
func TestGeocodeAddress_APIIDOnlyShownForHere(t *testing.T) {

	testCases := map[string]string{
		"HERE":            "here-app-id",
		"GEOAPIFY":        "",
		"GOOGLE-MAPS":     "",
		"OPEN-STREET-MAP": "",
		"":                "",
	}

	for provider, expectedAPIID := range testCases {

		t.Run("provider="+provider, func(t *testing.T) {

			config := NewGeocodeAddress().ManualConfig()
			value := newTestFormValue()

			require.NoError(t, config.SetURLValues(&value, url.Values{
				"data.provider": []string{provider},
				"data.apiID":    []string{"here-app-id"},
				"data.apiKey":   []string{"api-key"},
			}, testLookupProvider{}))

			require.Equal(t, expectedAPIID, value.GetMap("data").GetString("apiID"))
		})
	}
}

// TestGeocodeAddress_APIKeyNeedsAProvider confirms the credential field's show-if treats an
// empty provider as null, so a key posted without a provider is not written.
func TestGeocodeAddress_APIKeyNeedsAProvider(t *testing.T) {

	config := NewGeocodeAddress().ManualConfig()
	value := mapof.Any{"type": "", "active": false, "data": mapof.Any{"apiKey": "EXISTING-KEY"}}

	require.NoError(t, config.SetURLValues(&value, url.Values{
		"data.apiKey": []string{"NEW-KEY"},
	}, testLookupProvider{}))

	require.Equal(t, "EXISTING-KEY", value.GetMap("data").GetString("apiKey"), "an invisible field is never written")
}

// TestGeocodeAddress_OmittedAPIKeyIsCleared pins the other half of that rule. Once a provider
// is selected the credential field IS visible, and because it is declared Required:false the
// schema accepts the empty write that an omitted field produces.
func TestGeocodeAddress_OmittedAPIKeyIsCleared(t *testing.T) {

	config := NewGeocodeAddress().ManualConfig()
	value := mapof.Any{
		"type":   "",
		"active": false,
		"data":   mapof.Any{"provider": "GEOAPIFY", "apiKey": "EXISTING-KEY"},
	}

	require.NoError(t, config.SetURLValues(&value, url.Values{
		"data.provider": []string{"GEOAPIFY"},
	}, testLookupProvider{}))

	require.Empty(t, value.GetMap("data").GetString("apiKey"), "a partial post clears a visible field")
}

// TestGeocodeAddress_SavesAFullConfiguration confirms the whole form round-trips
func TestGeocodeAddress_SavesAFullConfiguration(t *testing.T) {

	config := NewGeocodeAddress().ManualConfig()
	value := newTestFormValue()

	require.NoError(t, config.SetURLValues(&value, url.Values{
		"type":          []string{model.ConnectionTypeGeocodeAddress},
		"data.provider": []string{"HERE"},
		"data.apiID":    []string{"here-app-id"},
		"data.apiKey":   []string{"here-api-key"},
		"active":        []string{"true"},
	}, testLookupProvider{}))

	data := value.GetMap("data")

	assert.Equal(t, model.ConnectionTypeGeocodeAddress, value.GetString("type"))
	assert.Equal(t, "HERE", data.GetString("provider"))
	assert.Equal(t, "here-app-id", data.GetString("apiID"))
	assert.Equal(t, "here-api-key", data.GetString("apiKey"))
	assert.True(t, value.GetBool("active"))
}

// TestGeocodeAddress_LatLongAreDeclaredButNotOffered pins two schema properties that no form
// element writes, left over from the Network geocoder's static-location mode.
func TestGeocodeAddress_LatLongAreDeclaredButNotOffered(t *testing.T) {

	config := NewGeocodeAddress().ManualConfig()
	paths := indexFormPaths(config)

	for _, path := range []string{"data.latitude", "data.longitude"} {

		_, declaredBySchema := config.Schema.GetElement(path)
		assert.True(t, declaredBySchema, "%s is in the schema", path)
		assert.False(t, paths[path], "%s has no form element", path)
	}
}

// TestGeocodeAddress_LifecycleIsANoOp confirms this provider only supplies a form
func TestGeocodeAddress_LifecycleIsANoOp(t *testing.T) {

	connection := newTestConnection()
	before := connection

	require.NoError(t, NewGeocodeAddress().BeforeSave(&connection, mapof.NewString()))
	require.NoError(t, NewGeocodeAddress().Connect(&connection, mapof.NewString(), "example.com"))
	require.NoError(t, NewGeocodeAddress().Refresh(&connection, mapof.NewString()))
	require.NoError(t, NewGeocodeAddress().Disconnect(&connection, mapof.NewString()))

	require.Equal(t, before, connection)
}
