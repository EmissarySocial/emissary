package geocoder

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestStatic_GeocodeNetwork pins the argument order, which is the whole risk in this
// type: NewStatic takes LATITUDE first, and geo.NewPoint takes LONGITUDE first.
func TestStatic_GeocodeNetwork(t *testing.T) {

	// Denver: latitude 39.7392, longitude -104.9903
	point, err := NewStatic(39.7392, -104.9903).GeocodeNetwork("8.8.8.8")

	require.Nil(t, err)
	require.Equal(t, 39.7392, point.Latitude)
	require.Equal(t, -104.9903, point.Longitude)
}

// TestStatic_GeocodeNetwork_IgnoresInput confirms that every IP resolves to the same
// fixed point, which is the entire contract of a Static geocoder.
func TestStatic_GeocodeNetwork_IgnoresInput(t *testing.T) {

	geocoder := NewStatic(51.5074, -0.1278)

	for _, ipAddress := range []string{"", "8.8.8.8", "127.0.0.1", "not-an-ip"} {

		point, err := geocoder.GeocodeNetwork(ipAddress)

		require.Nil(t, err)
		require.Equal(t, 51.5074, point.Latitude, "IP %q", ipAddress)
		require.Equal(t, -0.1278, point.Longitude, "IP %q", ipAddress)
	}
}

// TestStatic_GeocodeNetwork_Zero confirms that a Static geocoder configured at 0,0
// reports a zero Point, which callers test with IsZero.
func TestStatic_GeocodeNetwork_Zero(t *testing.T) {

	point, err := NewStatic(0, 0).GeocodeNetwork("8.8.8.8")

	require.Nil(t, err)
	require.True(t, point.IsZero())
}
