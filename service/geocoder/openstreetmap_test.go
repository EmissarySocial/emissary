package geocoder

import (
	"testing"
)

func TestOpenStreetMap(t *testing.T) {
	encoder := NewOpenStreetMap()
	testGeocodeAddress(t, encoder)
}
