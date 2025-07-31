package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/codingsince1985/geo-golang"
	"github.com/codingsince1985/geo-golang/arcgis"
	"github.com/codingsince1985/geo-golang/bing"
	"github.com/codingsince1985/geo-golang/google"
	"github.com/codingsince1985/geo-golang/openstreetmap"
	"github.com/codingsince1985/geo-golang/tomtom"
	"github.com/rs/zerolog/log"
)

type Geocode struct {
	hostname          string
	queue             *queue.Queue
	connectionService *Connection
}

func NewGeocode(hostname string, queue *queue.Queue, connectionService *Connection) Geocode {
	return Geocode{
		hostname:          hostname,
		queue:             queue,
		connectionService: connectionService,
	}
}

// Geocode will attempt to geocode all Places in the provided Stream
// and will return an error on any failure.
func (service Geocode) Geocode(session data.Session, stream *model.Stream) error {

	const location = "service.Geocode.GeocodeStream"

	geocoder := service.getGeocoder(session)

	for index := range stream.Places {

		// Get a pointer to the item in the slice
		place := &stream.Places[index]

		// If the place is already geocoded, then skip it
		if place.IsGeocoded {
			continue
		}

		// Try to geocode this place
		if err := service.geocode(geocoder, place); err != nil {
			return derp.Wrap(err, location, "Error geocoding place", place)
		}
	}

	return nil
}

// GeocodeAndQueue will attempt to geocode all Places in the provided Stream.
// If there is a failure, it will queue up a task to retry the geocode in 30 seconds.
func (service Geocode) GeocodeAndQueue(session data.Session, stream *model.Stream) error {

	const location = "service.Geocode.GeocodeAndQueue"

	// Try to Geocode all Places in this Stream
	if err := service.Geocode(session, stream); err == nil {
		return nil
	}

	// Queue up a geocode task for this place
	args := mapof.Any{
		"host":     service.hostname,
		"streamId": stream.StreamID,
	}

	task := queue.NewTask("Geocode", args, queue.WithPriority(4), queue.WithDelaySeconds(30))

	if err := service.queue.Publish(task); err != nil {
		return derp.Wrap(err, location, "Error publishing geocode task")
	}

	return nil
}

// geocode does the main work of geocoding a single place.
func (service Geocode) geocode(geocoder geo.Geocoder, place *model.Place) error {

	const location = "service.Connection.Geocode"
	place.ResetGeocode()

	// Try to get the coordinates for this place
	coordinates, err := geocoder.Geocode(place.FullAddress)

	log.Trace().Msg("Geocoding place: " + place.FullAddress)

	if err != nil {
		return derp.Wrap(err, location, "Error geocoding address", place)
	}

	place.IsGeocoded = true

	if coordinates == nil {
		return nil
	}

	// Set the coordinates
	place.Latitude = coordinates.Lat
	place.Longitude = coordinates.Lng

	// Try to find the address from the coordinates
	parsedAddress, err := geocoder.ReverseGeocode(coordinates.Lat, coordinates.Lng)

	if err != nil {
		return derp.Wrap(err, location, "Error reverse geocoding location", coordinates)
	}

	if parsedAddress == nil {
		return nil
	}

	// Set the address fields
	place.Street1 = parsedAddress.HouseNumber + " " + parsedAddress.Street
	place.Locality = parsedAddress.City
	place.Region = parsedAddress.State
	place.PostalCode = parsedAddress.Postcode
	place.Country = parsedAddress.Country

	return nil
}

// getGeocoder returns the geocoder configured for this domain.
// If none is configured, then the "free" OpenStreetMap geocoder is used.
func (service Geocode) getGeocoder(session data.Session) geo.Geocoder {

	// Get the geocoder connction config
	connection := model.NewConnection()

	if err := service.connectionService.LoadActiveByType(session, model.ConnectionTypeGeocoder, &connection); err != nil {
		// const location = "service.Geocode.getGeocoder"
		// derp.Report(derp.Wrap(err, location, "Error loading geocoder connection"))
		return openstreetmap.Geocoder()
	}

	switch connection.ProviderID {

	case model.ConnectionProviderArcGIS:
		return arcgis.Geocoder(connection.Data.GetString("apiKey"))

	case model.ConnectionProviderBing:
		return bing.Geocoder(connection.Data.GetString("apiKey"))

	case model.ConnectionProviderGoogleMaps:
		return google.Geocoder(connection.Data.GetString("apiKey"))

	case model.ConnectionProviderTomTom:
		return tomtom.Geocoder(connection.Data.GetString("apiKey"))
	}

	return openstreetmap.Geocoder()
}
