//gobuild:localonly

package geocoder

import (
	"testing"

	"github.com/benpate/geo"
	"github.com/stretchr/testify/require"
)

// To test, put a real API key here
var testGoogleMapsAPIKey string

func TestGoogleMaps_Address(t *testing.T) {
	encoder := NewGoogleMaps(testGoogleMapsAPIKey)
	testGeocodeAddress(t, encoder)
}

func TestGoogleMaps_Autocomplete(t *testing.T) {
	encoder := NewGoogleMaps(testGoogleMapsAPIKey)
	testAutocompleteAddress(t, encoder)
}

func TestGoogleMaps_Timezone(t *testing.T) {
	encoder := NewGoogleMaps(testGoogleMapsAPIKey)

	address := geo.Address{
		Longitude: -119.6822510,
		Latitude:  39.6034810,
	}
	err := encoder.GeocodeTimezone(&address)
	require.Nil(t, err)
	require.Equal(t, address.Timezone, "America/Los_Angeles")
}
