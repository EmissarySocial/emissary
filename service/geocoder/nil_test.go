package geocoder

import (
	"testing"

	"github.com/benpate/geo"
	"github.com/stretchr/testify/require"
)

// TestNil_GeocodeAddress confirms that the Nil geocoder echoes its query back as
// the formatted address, so a domain with no geocoder configured still round-trips.
func TestNil_GeocodeAddress(t *testing.T) {

	address, err := NewNil().GeocodeAddress("3317 E Colfax Ave, Denver, CO")

	require.Nil(t, err)
	require.Equal(t, "3317 E Colfax Ave, Denver, CO", address.Formatted)

	// Nothing else is invented: the Nil geocoder resolves no coordinates
	require.False(t, address.HasGeocode())
	require.False(t, address.HasAddress())
}

// TestNil_GeocodeAddress_Empty confirms an empty query produces an empty Address
// rather than an error.
func TestNil_GeocodeAddress_Empty(t *testing.T) {

	address, err := NewNil().GeocodeAddress("")

	require.Nil(t, err)
	require.Equal(t, "", address.Formatted)
	require.True(t, address.IsZero())
}

// TestNil_GeocodeNetwork confirms that the Nil geocoder resolves every IP address
// to the zero Point, and never to Null Island by way of a real-looking answer.
func TestNil_GeocodeNetwork(t *testing.T) {

	for _, ipAddress := range []string{"", "127.0.0.1", "8.8.8.8", "2601:280:5200::1", "not-an-ip"} {

		point, err := NewNil().GeocodeNetwork(ipAddress)

		require.Nil(t, err)
		require.True(t, point.IsZero(), "IP %q", ipAddress)
	}
}

// TestNil_AutocompleteAddress confirms that the Nil geocoder returns an empty,
// non-nil slice, so callers can range over it without a length check.
func TestNil_AutocompleteAddress(t *testing.T) {

	results, err := NewNil().AutocompleteAddress("Denver", geo.NewPoint(-104.9903, 39.7392))

	require.Nil(t, err)
	require.NotNil(t, results)
	require.Equal(t, 0, len(results))
}

// TestNil_GeocodeTimezone confirms that the Nil geocoder leaves the address untouched.
func TestNil_GeocodeTimezone(t *testing.T) {

	address := geo.Address{Formatted: "Denver, CO", Timezone: "America/Denver"}
	original := address

	require.Nil(t, NewNil().GeocodeTimezone(&address))
	require.Equal(t, original, address, "the Nil geocoder must not modify the address")
}

// TestNil_GeocodeTimezone_NilAddress documents that a nil address is safe, because
// every method body ignores its argument.
func TestNil_GeocodeTimezone_NilAddress(t *testing.T) {
	require.Nil(t, NewNil().GeocodeTimezone(nil))
}
