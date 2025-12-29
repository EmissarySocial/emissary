//gobuild:localonly

package geocoder

import (
	"testing"
)

// To test, put a real API key here
var testMaptilerAPIKey string

func TestMaptiler_Address(t *testing.T) {
	encoder := NewMaptiler(testMaptilerAPIKey)
	testGeocodeAddress(t, encoder)
}

func TestMaptiler_Autocomplete(t *testing.T) {
	encoder := NewMaptiler(testMaptilerAPIKey)
	testAutocompleteAddress(t, encoder)
}

/*
MAPTILER Network lookups are disabled, because they can
only return the location of the SERVER, and not the
location of the USER'S machine.  Sooo close :(

func TestMaptiler_Network(t *testing.T) {
	encoder := NewMaptiler(testMaptilerAPIKey)
	result, err := encoder.GeocodeNetwork("172.66.0.96")
	spew.Dump(result, err)
	// testGeocodeNetwork(t, encoder)
}
*/
