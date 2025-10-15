package geocode

import (
	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/mapof"
)

type IPAPICO struct {
	apiKey    string
	latitude  float64
	longitude float64
}

func NewIPAPICO(apiKey string, defaultLatitude float64, defaultLongitude float64) IPAPICO {
	return IPAPICO{
		apiKey:    apiKey,
		latitude:  defaultLatitude,
		longitude: defaultLongitude,
	}
}

func (geocoder IPAPICO) GeocodeIP(ipAddress string) (latitude float64, longitude float64, err error) {

	const location = "geocode.IPAPICO.GeocodeIP"

	result := mapof.NewAny()

	txn := remote.Get("https://ipapi.co/" + ipAddress + "/json/").Result(&result)

	if err := txn.Send(); err != nil {
		return geocoder.latitude, geocoder.longitude, derp.Wrap(err, location, "Error calling IPAPICO")
	}

	latitude = result.GetFloat("latitude")
	longitude = result.GetFloat("longitude")

	if latitude == 0 || longitude == 0 {
		return geocoder.latitude, geocoder.longitude, derp.InternalError(location, "No results found for this IP address", ipAddress)
	}

	return latitude, longitude, nil
}
