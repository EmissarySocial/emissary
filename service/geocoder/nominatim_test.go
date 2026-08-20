//go:build localonly

package geocoder

import (
	"testing"
)

// TestNominatim_Address exercises the Nominatim geocoder against the shared address suite
func TestNominatim_Address(t *testing.T) {
	encoder := NewNominatim("", "", "Emissary Test Suite", "localhost")
	testGeocodeAddress(t, encoder)
}

// TestNominatim_Autocomplete exercises the Nominatim geocoder against the shared autocomplete suite
func TestNominatim_Autocomplete(t *testing.T) {
	encoder := NewNominatim("", "", "Emissary Test Suite", "localhost")
	testAutocompleteAddress(t, encoder)
}
