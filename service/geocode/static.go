package geocode

type Static struct {
	latitude  float64
	longitude float64
}

func NewStatic(latitude float64, longitude float64) Geocoder {

	return Static{
		latitude:  latitude,
		longitude: longitude,
	}
}

func (geocoder Static) GeocodeIP(ipAddress string) (latitude float64, longitude float64, err error) {
	return geocoder.latitude, geocoder.longitude, nil
}
