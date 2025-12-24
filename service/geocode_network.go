package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service/geocoder"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/geo"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/turbine/queue"
)

type GeocodeNetwork struct {
	connectionService *Connection
	queue             *queue.Queue
	hostname          string
}

func NewGeocodeNetwork(connectionService *Connection, queue *queue.Queue, hostname string) GeocodeNetwork {
	return GeocodeNetwork{
		connectionService: connectionService,
		queue:             queue,
		hostname:          hostname,
	}
}

func (service GeocodeNetwork) Geocode(session data.Session, ipAddress string) (geo.Point, error) {

	const location = "service.GeocodeNetwork.Geocode"

	// Get IP geocoder
	geocoder := service.getGeocoder(session)

	// Get coordinates for this IP address
	point, err := geocoder.GeocodeNetwork(ipAddress)

	if err != nil {
		return geo.Point{}, derp.Wrap(err, location, "Error geocoding IP address", ipAddress)
	}

	// Success
	return point, nil
}

// getGeocoder returns the geocoder configured for this domain.
// If none is configured, then the "free" OpenStreetMap geocoder is used.
func (service GeocodeNetwork) getGeocoder(session data.Session) geocoder.NetworkGeocoder {

	const location = "service.GeocodeNetwork.getGeocoder"

	// Get the geocoder connction config
	connection := model.NewConnection()

	if err := service.connectionService.LoadActiveByType(session, model.ConnectionTypeGeocodeNetwork, &connection); err != nil {
		derp.Report(derp.Wrap(err, location, "Unable to load geocoder"))
	}

	latitude := connection.Data.GetString("latitude")   // nolint:scopeguard (readability)
	longitude := connection.Data.GetString("longitude") // nolint:scopeguard (readability)

	switch connection.Data.GetString("provider") {

	case "FREEIPAPICOM":
		return geocoder.NewFREEIPAPICOM(connection.Data.GetString("apiKey"))

	case "IPAPICOM":
		return geocoder.NewIPAPICOM(connection.Data.GetString("apiKey"))

	case "GEOAPIFY":
		return geocoder.NewGeoapify(connection.Data.GetString("apiKey"))

	case "STATIC":
		return geocoder.NewStatic(convert.Float(latitude), convert.Float(longitude))
	}

	// Default to static geocoder for Kansas City, MO
	return geocoder.NewStatic(
		39.0997,
		-94.5786,
	)
}
