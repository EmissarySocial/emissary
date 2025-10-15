package geocode

import (
	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/mapof"
)

type FREEIPAPICOM struct {
	apiKey    string
	latitude  float64
	longitude float64
}

func NewFREEIPAPICOM(apiKey string, defaultLatitude float64, defaultLongitude float64) FREEIPAPICOM {
	return FREEIPAPICOM{
		apiKey:    apiKey,
		latitude:  defaultLatitude,
		longitude: defaultLongitude,
	}
}

func (geocoder FREEIPAPICOM) GeocodeIP(ipAddress string) (latitude float64, longitude float64, err error) {

	const location = "geocode.FREEIPAPICOM.GeocodeIP"

	result := mapof.NewAny()

	txn := remote.Get("https://free.freeipapi.com/api/json/" + ipAddress).Result(&result)

	if err := txn.Send(); err != nil {
		return geocoder.latitude, geocoder.longitude, derp.Wrap(err, location, "Error calling FREEIPAPICOM")
	}

	latitude = result.GetFloat("latitude")
	longitude = result.GetFloat("longitude")

	if latitude == 0 || longitude == 0 {
		return geocoder.latitude, geocoder.longitude, derp.InternalError(location, "No results found for this IP address", ipAddress)
	}

	return latitude, longitude, nil
}
