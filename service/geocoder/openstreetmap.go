package geocoder

import (
	"github.com/benpate/derp"
	"github.com/benpate/geo"
	"github.com/codingsince1985/geo-golang/openstreetmap"
)

type OpenStreetMap struct{}

func NewOpenStreetMap() OpenStreetMap {
	return OpenStreetMap{}
}

func (geocoder OpenStreetMap) GeocodeAddress(address string) (point geo.Address, err error) {

	result, err := openstreetmap.Geocoder().Geocode(address)

	if err != nil {
		return geo.Address{}, derp.Wrap(err, "service.geocoder.OpenStreetMap.GeocodeAddress", "Unable to geocode address", address)
	}

	return geo.Address{
		Formatted: address,
		Longitude: result.Lng,
		Latitude:  result.Lat,
	}, nil
}
