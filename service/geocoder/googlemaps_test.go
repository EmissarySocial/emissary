//go:build localonly

package geocoder

import (
	"testing"

	"github.com/benpate/geo"
	"github.com/stretchr/testify/require"
)

// TestGoogleMaps_Address exercises the Google Maps geocoder against the shared address suite
func TestGoogleMaps_Address(t *testing.T) {
	encoder := NewGoogleMaps(testGoogleMapsAPIKey)
	testGeocodeAddress(t, encoder)
}

// TestGoogleMaps_Autocomplete exercises the Google Maps geocoder against the shared autocomplete suite
func TestGoogleMaps_Autocomplete(t *testing.T) {
	encoder := NewGoogleMaps(testGoogleMapsAPIKey)
	testAutocompleteAddress(t, encoder)
}

// TestGoogleMaps_Timezone exercises the Google Maps geocoder against the shared timezone suite
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
