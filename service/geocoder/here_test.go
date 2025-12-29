//go:build localonly

package geocoder

import (
	"testing"
)

func TestHere_Address(t *testing.T) {
	encoder := NewHere(testHereAPIID, testHereAPIKey)
	testGeocodeAddress(t, encoder)
}

func TestHere_Autocomplete(t *testing.T) {
	encoder := NewHere(testHereAPIID, testHereAPIKey)
	testAutocompleteAddress(t, encoder)
}

func TestHere_Timezone(t *testing.T) {
	encoder := NewHere(testHereAPIID, testHereAPIKey)
	testGeocodeTimezone(t, encoder)
}
