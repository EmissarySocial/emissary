package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service/geocoder"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

type GeocodeAddress struct {
	hostname          string
	queue             *queue.Queue
	connectionService *Connection
	timezoneService   GeocodeTimezone
}

func NewGeocodeAddress(hostname string, queue *queue.Queue, connectionService *Connection, timezoneService GeocodeTimezone) GeocodeAddress {
	return GeocodeAddress{
		hostname:          hostname,
		queue:             queue,
		connectionService: connectionService,
		timezoneService:   timezoneService,
	}
}

// GeocodeAndQueue will attempt to geocode the Location in the provided Stream.
// If there is a failure, it will queue up a task to retry the geocode in 30 seconds.
func (service GeocodeAddress) GeocodeAndQueue(session data.Session, stream *model.Stream) error {

	// Try to GeocodeAddress all Places in this Stream
	if err := service.Geocode(session, stream); err != nil {

		// If there is an error, then enqueue a Task to retru in 30 seconds
		service.queue.NewTask(
			"GeocodeAddress",
			mapof.Any{
				"host":     service.hostname,
				"streamId": stream.StreamID,
			},
			queue.WithDelaySeconds(30),
		)
	}

	return nil
}

// Geocode tries to geocode the Location in the provided Stream
// and will return an error on any failure.
func (service GeocodeAddress) Geocode(session data.Session, stream *model.Stream) error {

	const location = "service.GeocodeAddress.GeocodeAddress"

	// RULE: If the Stream has already been geocoded, then exit
	if stream.Location.HasGeocode() {
		return nil
	}

	// Find the Geocoder configured for this Domain
	geocoder := service.getGeocoder(session)

	// Try to get the coordinates for this place
	address, err := geocoder.GeocodeAddress(stream.Location.Formatted)

	if err != nil {
		return derp.Wrap(err, location, "Error geocoding address", stream.Location.Formatted)
	}

	// Locate the timezone, too, if we don't already have it.
	if err := service.timezoneService.Geocode(session, &address); err != nil {
		return derp.Wrap(err, location, "Unable to get timezone for location", stream.Location)
	}

	// Populate the Stream with the newly geocoded address
	stream.Location = address
	return nil
}

// getGeocoder returns the geocoder configured for this domain.
// If none is configured, then the "free" OpenStreetMap geocoder is used.
func (service GeocodeAddress) getGeocoder(session data.Session) geocoder.AddressGeocoder {

	const location = "service.GeocodeAutocommplete.getGeocoder"

	// Get the geocoder connction config
	connection := model.NewConnection()

	if err := service.connectionService.LoadActiveByType(session, model.ConnectionTypeGeocodeAddress, &connection); err != nil {
		derp.Report(derp.Wrap(err, location, "Unable to load geocoder"))
	}

	apiKey := connection.Data.GetString("apiKey") // nolint:scopeguard

	switch connection.Data.GetString("provider") {

	case "HERE":
		return geocoder.NewHere(connection.Data.GetString("apiID"), apiKey)

	case "GEOAPIFY":
		return geocoder.NewGeoapify(apiKey)

	case "GEOCODIO":
		return geocoder.NewGeocodio(apiKey)

	case "GOOGLE-MAPS":
		return geocoder.NewGoogleMaps(apiKey)

	case "MAPTILER":
		return geocoder.NewMaptiler(apiKey)

	case "OPEN-STREET-MAP":
		return geocoder.NewOpenStreetMap()
	}

	return geocoder.NewOpenStreetMap()
}
