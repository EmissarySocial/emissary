package geocoder

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAPIKeys_Declared references every live-test credential, so a normal build does not
// report them as unused. The localonly build assigns them in apikeys_test.go.
func TestAPIKeys_Declared(t *testing.T) {

	credentials := map[string]string{
		"Geoapify":   testGeoapifyAPIKey,
		"Geocodio":   testGeocodioAPIKey,
		"GoogleMaps": testGoogleMapsAPIKey,
		"HERE id":    testHereAPIID,
		"HERE key":   testHereAPIKey,
		"Maptiler":   testMaptilerAPIKey,
	}

	require.Equal(t, 6, len(credentials), "every localonly credential must be listed here")

	// Report which providers cannot run their live tests, without failing either build
	for name, value := range credentials {
		if value == "" {
			t.Logf("%s credential is unset -- its localonly tests cannot run", name)
		}
	}
}
