package geocoder

import (
	"testing"
)

// To test, put a real API key here
var testGoogleMapsAPIKey = "AIzaSyBAmetmVwMtYcaEI5IQr1wiDKEZO7ceIOE"

func TestGoogleMaps_Address(t *testing.T) {
	encoder := NewGoogleMaps(testGoogleMapsAPIKey)
	testGeocodeAddress(t, encoder)
}

func TestGoogleMaps_Autocomplete(t *testing.T) {
	encoder := NewGoogleMaps(testGoogleMapsAPIKey)
	testAutocompleteAddress(t, encoder)
}
