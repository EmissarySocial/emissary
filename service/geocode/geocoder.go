package geocode

type Geocoder interface {
	GeocodeIP(ipAddress string) (latitude float64, longitude float64, err error)
}
