package geocoder

import (
	"github.com/benpate/derp"
	"github.com/benpate/geo"
	"github.com/codingsince1985/geo-golang/openstreetmap"
)

// OpenStreetMap geocodes addresses using the public OpenStreetMap Nominatim service
type OpenStreetMap struct{}

// NewOpenStreetMap returns a OpenStreetMap geocoder
func NewOpenStreetMap() OpenStreetMap {
	return OpenStreetMap{}
}

// GeocodeAddress resolves a free-text address into a geo.Address. Implements the AddressGeocoder interface.
func (geocoder OpenStreetMap) GeocodeAddress(address string) (point geo.Address, err error) {

	result, err := openstreetmap.Geocoder().Geocode(address)

	if err != nil {
		return geo.Address{}, derp.Wrap(err, "service.geocoder.OpenStreetMap.GeocodeAddress", "Geocoding address", address)
	}

	return geo.Address{
		Formatted: address,
		Longitude: result.Lng,
		Latitude:  result.Lat,
	}, nil
}
