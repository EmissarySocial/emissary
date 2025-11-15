package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service/geocoder"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/geo"
)

type GeocodeTimezone struct {
	connectionService *Connection
	hostname          string
}

func NewGeocodeTimezone(connectionService *Connection, hostname string) GeocodeTimezone {
	return GeocodeTimezone{
		connectionService: connectionService,
		hostname:          hostname,
	}
}

// Geocode tries to geocode the Location in the provided Stream
// and will return an error on any failure.
func (service GeocodeTimezone) Geocode(session data.Session, address *geo.Address) error {

	const location = "service.GeocodeTimezone.GeocodeTimezone"

	// If the address already has a timezone, then don't fetch it again
	if address.Timezone != "" {
		return nil
	}

	// Find the Geocoder configured for this Domain
	geocoder := service.getGeocoder(session)

	// Try to get the coordinates for this place
	if err := geocoder.GeocodeTimezone(address); err != nil {
		return derp.Wrap(err, location, "Unable to retrieve timezone for address", address)
	}

	return nil
}

// getGeocoder returns the geocoder configured for this domain.
// If none is configured, then the "free" OpenStreetMap geocoder is used.
func (service GeocodeTimezone) getGeocoder(session data.Session) geocoder.TimezoneGeocoder {

	const location = "service.GeocodeAutocommplete.getGeocoder"

	// Get the geocoder connction config
	connection := model.NewConnection()

	if err := service.connectionService.LoadActiveByType(session, model.ConnectionTypeGeocodeTimezone, &connection); err != nil {
		derp.Report(derp.Wrap(err, location, "Unable to load geocoder"))
	}

	switch connection.Data.GetString("provider") {

	case "HERE":
		return geocoder.NewHere(connection.Data.GetString("apiID"), connection.Data.GetString("apiKey"))

	case "GEOAPIFY":
		return geocoder.NewGeoapify(connection.Data.GetString("apiKey"))

	case "GEOCODIO":
		return geocoder.NewGeocodio(connection.Data.GetString("apiKey"))

	case "GOOGLE":
		return geocoder.NewGoogleMaps(connection.Data.GetString("apiKey"))

	}

	return geocoder.NewNil()
}
