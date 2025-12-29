//go:build localonly

package geocoder

import (
	"testing"
)

// To test, put a real API key here
var testGeoapifyAPIKey string

func TestGeoapify_Address(t *testing.T) {
	encoder := NewGeoapify(testGeoapifyAPIKey)
	testGeocodeAddress(t, encoder)
}

func TestGeoapify_Autocomplete(t *testing.T) {
	encoder := NewGeoapify(testGeoapifyAPIKey)
	testAutocompleteAddress(t, encoder)
}

func TestGeoapify_Network(t *testing.T) {
	encoder := NewGeoapify(testGeoapifyAPIKey)
	testGeocodeNetwork(t, encoder)
}

func TestGeoapify_Timezone(t *testing.T) {
	encoder := NewGeoapify(testGeoapifyAPIKey)
	testGeocodeTimezone(t, encoder)
}
