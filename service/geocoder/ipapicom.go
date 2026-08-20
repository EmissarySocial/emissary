package geocoder

import (
	"github.com/benpate/derp"
	"github.com/benpate/geo"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/mapof"
)

// IPAPICOM geocodes IP addresses using the ip-api.com service
type IPAPICOM struct {
	apiKey string
}

// NewIPAPICOM returns a IPAPICOM geocoder
func NewIPAPICOM(apiKey string) IPAPICOM {
	return IPAPICOM{
		apiKey: apiKey,
	}
}

// GeocodeNetwork resolves an IP address into approximate coordinates. Implements the NetworkGeocoder interface.
func (geocoder IPAPICOM) GeocodeNetwork(ipAddress string) (point geo.Point, err error) {

	const location = "geocode.IPAPICOM.GeocodeNetwork"

	result := mapof.NewAny()

	txn := remote.Get("http://ip-api.com/json/" + ipAddress).Result(&result)

	if err := txn.Send(); err != nil {
		return geo.Point{}, derp.Wrap(err, location, "Calling IPAPICOM")
	}

	latitude := result.GetFloat("lat")
	longitude := result.GetFloat("lon")

	return geo.NewPoint(longitude, latitude), nil
}
