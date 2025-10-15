package geocode

import (
	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/mapof"
)

type IPAPICOM struct {
	apiKey    string
	latitude  float64
	longitude float64
}

func NewIPAPICOM(apiKey string, defaultLatitude float64, defaultLongitude float64) IPAPICOM {
	return IPAPICOM{
		apiKey:    apiKey,
		latitude:  defaultLatitude,
		longitude: defaultLongitude,
	}
}

func (geocoder IPAPICOM) GeocodeIP(ipAddress string) (latitude float64, longitude float64, err error) {

	const location = "geocode.IPAPICOM.GeocodeIP"

	result := mapof.NewAny()

	txn := remote.Get("http://ip-api.com/json/" + ipAddress).Result(&result)

	if err := txn.Send(); err != nil {
		return geocoder.latitude, geocoder.longitude, derp.Wrap(err, location, "Error calling IPAPICOM")
	}

	latitude = result.GetFloat("lat")
	longitude = result.GetFloat("lon")

	if latitude == 0 || longitude == 0 {
		return geocoder.latitude, geocoder.longitude, derp.InternalError(location, "No results found for this IP address", ipAddress)
	}

	return latitude, longitude, nil
}
