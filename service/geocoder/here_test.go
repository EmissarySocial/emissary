//go:build localonly

package geocoder

import (
	"testing"
)

// TestHere_Address exercises the HERE geocoder against the shared address suite
func TestHere_Address(t *testing.T) {
	encoder := NewHere(testHereAPIID, testHereAPIKey)
	testGeocodeAddress(t, encoder)
}

// TestHere_Autocomplete exercises the HERE geocoder against the shared autocomplete suite
func TestHere_Autocomplete(t *testing.T) {
	encoder := NewHere(testHereAPIID, testHereAPIKey)
	testAutocompleteAddress(t, encoder)
}

// TestHere_Timezone exercises the HERE geocoder against the shared timezone suite
func TestHere_Timezone(t *testing.T) {
	encoder := NewHere(testHereAPIID, testHereAPIKey)
	testGeocodeTimezone(t, encoder)
}
