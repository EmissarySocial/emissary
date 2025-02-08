package service

import (
	"testing"

	"github.com/codingsince1985/geo-golang/openstreetmap"
	"github.com/davecgh/go-spew/spew"
)

func TestOpenStreetMap(t *testing.T) {

	coder := openstreetmap.Geocoder()

	coordinates, err := coder.Geocode("1600 Amphitheatre Parkway, Mountain View, CA 94043")

	spew.Dump(coordinates, err)
}
