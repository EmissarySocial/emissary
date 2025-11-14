package geocoder

import (
	"testing"
)

func TestNominatim_Address(t *testing.T) {
	encoder := NewNominatim("", "", "Emissary Test Suite", "localhost")
	testGeocodeAddress(t, encoder)
}
