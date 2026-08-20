package geocoder

import (
	"github.com/benpate/derp"
	"github.com/benpate/geo"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/mapof"
)

// FREEIPAPICOM geocodes IP addresses using the freeipapi.com service
type FREEIPAPICOM struct {
	apiKey string
}

// NewFREEIPAPICOM returns a FREEIPAPICOM geocoder
func NewFREEIPAPICOM(apiKey string) FREEIPAPICOM {
	return FREEIPAPICOM{
		apiKey: apiKey,
	}
}

// GeocodeNetwork resolves an IP address into approximate coordinates. Implements the NetworkGeocoder interface.
func (geocoder FREEIPAPICOM) GeocodeNetwork(ipAddress string) (point geo.Point, err error) {

	const location = "geocode.FREEIPAPICOM.GeocodeNetwork"

	result := mapof.NewAny()

	txn := remote.Get("https://free.freeipapi.com/api/json/" + ipAddress).Result(&result)

	if err := txn.Send(); err != nil {
		return geo.NewPoint(0, 0), derp.Wrap(err, location, "Calling FREEIPAPICOM")
	}

	longitude := result.GetFloat("longitude")
	latitude := result.GetFloat("latitude")

	if (longitude == 0) || (latitude == 0) {
		return geo.NewPoint(0, 0), derp.Internal(location, "No results found for this IP address", ipAddress)
	}

	return geo.NewPoint(longitude, latitude), nil
}
