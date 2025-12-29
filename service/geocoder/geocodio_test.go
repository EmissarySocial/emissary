package geocoder

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Geocodio only works in the US, so this international test is failing.
func TestGeocodio_Address(t *testing.T) {
	require.NotEmpty(t, testGeocodioAPIKey)

	// Geocodio only works int the US, so this international test fails.
	// encoder := NewGeocodio(testGeocodioAPIKey)
	// testGeocodeAddress(t, encoder)
}
