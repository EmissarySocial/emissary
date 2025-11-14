package geocoder

import "github.com/benpate/geo"

type Static struct {
	latitude  float64
	longitude float64
}

func NewStatic(latitude float64, longitude float64) Static {

	return Static{
		latitude:  latitude,
		longitude: longitude,
	}
}

func (geocoder Static) GeocodeNetwork(ipAddress string) (point geo.Point, err error) {
	return geo.NewPoint(geocoder.longitude, geocoder.latitude), nil
}
