package geocode

import (
	"testing"

	"github.com/benpate/derp"
	"github.com/davecgh/go-spew/spew"
)

func TestFREEIPAPICOM(t *testing.T) {

	geocoder := NewFREEIPAPICOM("", 9, 9)

	latitude, longitude, err := geocoder.GeocodeIP("216.224.124.125")

	spew.Dump(latitude, longitude)
	derp.Report(err)
}
