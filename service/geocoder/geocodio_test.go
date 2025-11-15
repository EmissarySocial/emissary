package geocoder

import "testing"

var testGeocodioAPIKey string

func TestGeocodio_Address(t *testing.T) {
	encoder := NewGeocodio(testGeocodioAPIKey)
	testGeocodeAddress(t, encoder)
}
