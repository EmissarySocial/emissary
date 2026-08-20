//go:build localonly

package geocoder

import (
	"testing"
)

// TestGeoapify_Address exercises the Geoapify geocoder against the shared address suite
func TestGeoapify_Address(t *testing.T) {
	encoder := NewGeoapify(testGeoapifyAPIKey)
	testGeocodeAddress(t, encoder)
}

// TestGeoapify_Autocomplete exercises the Geoapify geocoder against the shared autocomplete suite
func TestGeoapify_Autocomplete(t *testing.T) {
	encoder := NewGeoapify(testGeoapifyAPIKey)
	testAutocompleteAddress(t, encoder)
}

// TestGeoapify_Network exercises the Geoapify geocoder against the shared network suite
func TestGeoapify_Network(t *testing.T) {
	encoder := NewGeoapify(testGeoapifyAPIKey)
	testGeocodeNetwork(t, encoder)
}

// TestGeoapify_Timezone exercises the Geoapify geocoder against the shared timezone suite
func TestGeoapify_Timezone(t *testing.T) {
	encoder := NewGeoapify(testGeoapifyAPIKey)
	testGeocodeTimezone(t, encoder)
}
