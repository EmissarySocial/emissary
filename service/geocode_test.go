package service

import (
	"math"
	"testing"

	"github.com/codingsince1985/geo-golang/openstreetmap"
	"github.com/stretchr/testify/require"
)

func TestOpenStreetMap(t *testing.T) {

	coder := openstreetmap.Geocoder()

	coordinates, err := coder.Geocode("1600 Amphitheatre Parkway, Mountain View, CA 94043")

	require.NoError(t, err)
	require.Equal(t, float64(3742), math.Floor(coordinates.Lat*100))
	require.Equal(t, float64(-12209), math.Floor(coordinates.Lng*100))
}
