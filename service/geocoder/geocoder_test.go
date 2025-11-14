package geocoder

import (
	"math"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
)

type testLocation struct {
	query     string
	longitude float64
	latitude  float64
}

func testGeocodeAddress(t *testing.T, geocoder AddressGeocoder) {

	run := func(location testLocation) {
		address, err := geocoder.GeocodeAddress(location.query)
		spew.Dump(address)

		require.Nil(t, err)
		require.True(t, closeEnough(t, location.longitude, address.Longitude))
		require.True(t, closeEnough(t, location.latitude, address.Latitude))
	}

	// Test Result from Google
	run(testLocation{
		query:     "1600 Amphitheatre Pkwy, Mountain View, CA 94043, USA",
		longitude: -122.0843428,
		latitude:  37.4222804,
	})

	// Test Result from Geoapify.com
	run(testLocation{
		query:     "38 Upper Montagu Street, Westminster W1H 1LJ, United Kingdom",
		longitude: -0.16030636023550826,
		latitude:  51.52016005,
	})

	// Colorado Capitol Building
	run(testLocation{
		query:     "200 E Colfax Ave, Denver, CO 80203 United States",
		longitude: -104.98484674529767,
		latitude:  39.739211499999996,
	})
}

func testGeocodeNetwork(t *testing.T, geocoder NetworkGeocoder) {

	run := func(location testLocation) {
		point, err := geocoder.GeocodeNetwork(location.query)

		require.Nil(t, err)
		require.NotZero(t, point.Latitude)
		require.NotZero(t, point.Longitude)

		// require.True(t, closeEnough(t, location.longitude, point.Longitude))
		// require.True(t, closeEnough(t, location.latitude, point.Latitude))
	}

	// Test Result from Geoapify.com
	run(testLocation{
		query:     "216.224.124.125",
		longitude: -106.651,
		latitude:  35.0845,
	})
}

func testAutocompleteAddress(t *testing.T, geocoder AddressAutocompleter) {

	run := func(location string) {
		addresses, err := geocoder.AutocompleteAddress(location)

		require.Nil(t, err)
		require.NotZero(t, addresses.Length())
	}

	// Test Result from Geoapify.com
	run("200 E Colfax Ave, Denver, CO 80203, United States")
	run("Pearl Street, Boulder")

	// Test Value from Google
	run("1600 Amphitheatre Pkwy, Mountain View, CA 94043, USA")
}

// closeEnough compares two floats, returning TRUE if they
// are equal within 3 decimal points
func closeEnough(t *testing.T, a float64, b float64) bool {

	a = math.Floor(a * float64(10^3))
	b = math.Floor(b * float64(10^3))

	if a == b {
		return true
	}

	t.Log(a, b)

	return false
}
