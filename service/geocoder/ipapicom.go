package geocoder

import (
	"github.com/benpate/derp"
	"github.com/benpate/geo"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/mapof"
)

type IPAPICOM struct {
	apiKey string
}

func NewIPAPICOM(apiKey string) IPAPICOM {
	return IPAPICOM{
		apiKey: apiKey,
	}
}

func (geocoder IPAPICOM) GeocodeNetwork(ipAddress string) (point geo.Point, err error) {

	const location = "geocode.IPAPICOM.GeocodeNetwork"

	result := mapof.NewAny()

	txn := remote.Get("http://ip-api.com/json/" + ipAddress).Result(&result)

	if err := txn.Send(); err != nil {
		return geo.Point{}, derp.Wrap(err, location, "Error calling IPAPICOM")
	}

	latitude := result.GetFloat("lat")
	longitude := result.GetFloat("lon")

	return geo.NewPoint(longitude, latitude), nil
}
