package geocoder

import "github.com/benpate/geo"

// Static is a NetworkGeocoder that answers every query with one fixed coordinate
type Static struct {
	latitude  float64
	longitude float64
}

// NewStatic returns a Static geocoder
func NewStatic(latitude float64, longitude float64) Static {

	return Static{
		latitude:  latitude,
		longitude: longitude,
	}
}

// GeocodeNetwork resolves an IP address into approximate coordinates. Implements the NetworkGeocoder interface.
func (geocoder Static) GeocodeNetwork(ipAddress string) (point geo.Point, err error) {
	return geo.NewPoint(geocoder.longitude, geocoder.latitude), nil
}
