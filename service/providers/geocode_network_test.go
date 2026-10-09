package providers

import (
	"net/url"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGeocodeNetwork_OfferedProviders pins the service list, STATIC included. STATIC is the
// only value that trades a credential for a fixed pair of coordinates.
func TestGeocodeNetwork_OfferedProviders(t *testing.T) {

	config := NewGeocodeNetwork().ManualConfig()

	require.Equal(t, []string{"GEOAPIFY", "FREEIPAPI", "IPAPICOM", "STATIC"},
		selectOptionValues(config, "data.provider"))
}

// TestGeocodeNetwork_TypeAndProviderConstantsDiffer pins the one Connection constant pair that
// is not identical. This form writes the Type constant, so the Provider constant must not leak in.
func TestGeocodeNetwork_TypeAndProviderConstantsDiffer(t *testing.T) {

	require.Equal(t, "GEOCODER-NETWORK", model.ConnectionTypeGeocodeNetwork)
	require.Equal(t, "GEOCODE-NETWORK", model.ConnectionProviderGeocodeNetwork)
	require.NotEqual(t, model.ConnectionTypeGeocodeNetwork, model.ConnectionProviderGeocodeNetwork)

	config := NewGeocodeNetwork().ManualConfig()
	require.Equal(t, model.ConnectionTypeGeocodeNetwork, hiddenTypeValue(config))
}

// TestGeocodeNetwork_StaticModeSwapsTheCredentialForCoordinates confirms the three show-if
// rules that split a credentialed lookup service from a hard-coded location.
func TestGeocodeNetwork_StaticModeSwapsTheCredentialForCoordinates(t *testing.T) {

	testCases := map[string]struct {
		provider          string
		expectedAPIKey    string
		expectedLatitude  string
		expectedLongitude string
	}{
		"a lookup service takes a key": {"GEOAPIFY", "api-key", "", ""},
		"FreeIPAPI takes a key":        {"FREEIPAPI", "api-key", "", ""},
		"STATIC takes coordinates":     {"STATIC", "", "39.7392", "-104.9903"},
		"no provider takes nothing":    {"", "", "", ""},
	}

	for name, testCase := range testCases {

		t.Run(name, func(t *testing.T) {

			config := NewGeocodeNetwork().ManualConfig()
			value := newTestFormValue()

			require.NoError(t, config.SetURLValues(&value, url.Values{
				"data.provider":  []string{testCase.provider},
				"data.apiKey":    []string{"api-key"},
				"data.latitude":  []string{"39.7392"},
				"data.longitude": []string{"-104.9903"},
			}, testLookupProvider{}))

			data := value.GetMap("data")
			assert.Equal(t, testCase.expectedAPIKey, data.GetString("apiKey"))
			assert.Equal(t, testCase.expectedLatitude, data.GetString("latitude"))
			assert.Equal(t, testCase.expectedLongitude, data.GetString("longitude"))
		})
	}
}

// TestGeocodeNetwork_ActiveIsWrittenFromInsideAHiddenContainer pins a trap in the form layout.
// The Enable toggle sits inside a container gated on a provider being chosen, but a container's
// show-if does not gate its children's writes, so active saves with no provider at all.
func TestGeocodeNetwork_ActiveIsWrittenFromInsideAHiddenContainer(t *testing.T) {

	config := NewGeocodeNetwork().ManualConfig()
	value := newTestFormValue()

	require.NoError(t, config.SetURLValues(&value, url.Values{
		"active": []string{"true"},
	}, testLookupProvider{}))

	require.True(t, value.GetBool("active"), "the toggle is written even while its container is hidden")
	require.Empty(t, value.GetMap("data").GetString("provider"), "no provider was selected")
}

// TestGeocodeNetwork_CoordinatesAreStoredAsStrings pins the schema choice, because a caller
// reading these as floats must convert them rather than type-asserting.
func TestGeocodeNetwork_CoordinatesAreStoredAsStrings(t *testing.T) {

	config := NewGeocodeNetwork().ManualConfig()
	value := newTestFormValue()

	require.NoError(t, config.SetURLValues(&value, url.Values{
		"data.provider":  []string{"STATIC"},
		"data.latitude":  []string{"39.7392"},
		"data.longitude": []string{"-104.9903"},
	}, testLookupProvider{}))

	data := value.GetMap("data")
	require.IsType(t, "", data["latitude"])
	require.IsType(t, "", data["longitude"])
}

// TestGeocodeNetwork_LifecycleIsANoOp confirms this provider only supplies a form
func TestGeocodeNetwork_LifecycleIsANoOp(t *testing.T) {

	connection := newTestConnection()
	before := connection

	require.NoError(t, NewGeocodeNetwork().BeforeSave(&connection, mapof.NewString()))
	require.NoError(t, NewGeocodeNetwork().Connect(&connection, mapof.NewString(), "example.com"))
	require.NoError(t, NewGeocodeNetwork().Refresh(&connection, mapof.NewString()))
	require.NoError(t, NewGeocodeNetwork().Disconnect(&connection, mapof.NewString()))

	require.Equal(t, before, connection)
}
