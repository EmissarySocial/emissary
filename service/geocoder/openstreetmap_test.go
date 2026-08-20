//go:build localonly

package geocoder

import (
	"testing"
)

// TestOpenStreetMap exercises the OpenStreetMap geocoder against the shared address suite
func TestOpenStreetMap(t *testing.T) {
	encoder := NewOpenStreetMap()
	testGeocodeAddress(t, encoder)
}
