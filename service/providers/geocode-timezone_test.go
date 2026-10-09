package providers

import (
	"net/url"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGeocodeTimezone_OfferedProviders pins the service list. Unlike the address forms this one
// spells Google as GOOGLE rather than GOOGLE-MAPS.
func TestGeocodeTimezone_OfferedProviders(t *testing.T) {

	config := NewGeocodeTimezone().ManualConfig()
	offered := selectOptionValues(config, "data.provider")

	require.Equal(t, []string{"GEOAPIFY", "HERE", "GOOGLE", "GEOCODIO"}, offered)
	require.NotContains(t, offered, "GOOGLE-MAPS", "the address forms use the longer spelling")
}

// TestGeocodeTimezone_SchemaEnumAndWrittenTypeAgree confirms the schema names the type with the
// Provider constant while the hidden element writes the Type constant, and that the two are equal.
func TestGeocodeTimezone_SchemaEnumAndWrittenTypeAgree(t *testing.T) {

	require.Equal(t, model.ConnectionProviderGeocodeTimezone, model.ConnectionTypeGeocodeTimezone)

	config := NewGeocodeTimezone().ManualConfig()
	require.Equal(t, model.ConnectionTypeGeocodeTimezone, hiddenTypeValue(config))

	element, exists := config.Schema.GetElement("type")
	require.True(t, exists)

	stringElement, isString := element.(schema.String)
	require.True(t, isString)
	require.Equal(t, []string{model.ConnectionProviderGeocodeTimezone}, stringElement.Enum)
}

// TestGeocodeTimezone_APIKeyHasNoProviderGate pins the difference from the other geocoders:
// the API Key field carries no show-if, so it is written whether or not a provider is chosen.
func TestGeocodeTimezone_APIKeyHasNoProviderGate(t *testing.T) {

	config := NewGeocodeTimezone().ManualConfig()
	value := newTestFormValue()

	require.NoError(t, config.SetURLValues(&value, url.Values{
		"data.apiKey": []string{"api-key"},
	}, testLookupProvider{}))

	require.Equal(t, "api-key", value.GetMap("data").GetString("apiKey"))
	require.Empty(t, value.GetMap("data").GetString("provider"))
}

// TestGeocodeTimezone_APIIDOnlyShownForHere confirms the one gated field still behaves
func TestGeocodeTimezone_APIIDOnlyShownForHere(t *testing.T) {

	testCases := map[string]string{
		"HERE":     "here-app-id",
		"GEOAPIFY": "",
		"GOOGLE":   "",
		"GEOCODIO": "",
		"":         "",
	}

	for provider, expectedAPIID := range testCases {

		t.Run("provider="+provider, func(t *testing.T) {

			config := NewGeocodeTimezone().ManualConfig()
			value := newTestFormValue()

			require.NoError(t, config.SetURLValues(&value, url.Values{
				"data.provider": []string{provider},
				"data.apiID":    []string{"here-app-id"},
			}, testLookupProvider{}))

			assert.Equal(t, expectedAPIID, value.GetMap("data").GetString("apiID"))
		})
	}
}

// TestGeocodeTimezone_LifecycleIsANoOp confirms this provider only supplies a form
func TestGeocodeTimezone_LifecycleIsANoOp(t *testing.T) {

	connection := newTestConnection()
	before := connection

	require.NoError(t, NewGeocodeTimezone().BeforeSave(&connection, mapof.NewString()))
	require.NoError(t, NewGeocodeTimezone().Connect(&connection, mapof.NewString(), "example.com"))
	require.NoError(t, NewGeocodeTimezone().Refresh(&connection, mapof.NewString()))
	require.NoError(t, NewGeocodeTimezone().Disconnect(&connection, mapof.NewString()))

	require.Equal(t, before, connection)
}
