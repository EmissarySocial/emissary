package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service/geocoder"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/geo"
	"github.com/benpate/rosetta/sliceof"
)

type GeocodeAutocomplete struct {
	connectionService *Connection
	hostname          string
}

func NewGeocodeAutocomplete(connectionService *Connection, hostname string) GeocodeAutocomplete {
	return GeocodeAutocomplete{
		connectionService: connectionService,
		hostname:          hostname,
	}
}

// Search retrieves all values matching the query parameter, as returned by the available search service.
func (service GeocodeAutocomplete) Search(session data.Session, query string, referer string) (sliceof.Object[geo.Address], error) {

	const location = "service.GeocodeAutocomplete.GeocodeAutocomplete"

	geocoder := service.getGeocoder(session)
	result, err := geocoder.AutocompleteAddress(query)

	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to retrieve search results")
	}

	return result, nil
}

// getGeocoder returns the provider configured for this domain.
// If none is configured, then the "free" OpenStreetMap provider is used.
func (service GeocodeAutocomplete) getGeocoder(session data.Session) AddressAutocompleter {

	const location = "service.GeocodeAutocommplete.getGeocoder"

	// Get the provider connction config
	connection := model.NewConnection()

	if err := service.connectionService.LoadActiveByType(session, model.ConnectionTypeGeocodeAutocomplete, &connection); err != nil {
		derp.Report(derp.Wrap(err, location, "Unable to load Autocomplete connection"))
	}

	switch connection.Data.GetString("provider") {

	case "GEOAPIFY":
		return geocoder.NewGeoapify(connection.Data.GetString("apiKey"))

	case "MAPTILER":
		return geocoder.NewMaptiler(connection.Data.GetString("apiKey"))

	case "NOMINATIM":
		return geocoder.NewNominatim(
			connection.Data.GetString("serverUrl"),
			connection.Data.GetString("apiKey"),
			"Emissary (emissary.dev)",
			service.hostname,
		)
	}

	return geocoder.NewNil()
}
