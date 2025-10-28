package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service/geosearch"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/sliceof"
)

type Geosearch struct {
	connectionService *Connection
	hostname          string
}

func NewGeosearch(connectionService *Connection, hostname string) Geosearch {
	return Geosearch{
		connectionService: connectionService,
		hostname:          hostname,
	}
}

// Search retrieves all values matching the query parameter, as returned by the available search service.
func (service Geosearch) Search(session data.Session, query string, referer string) (sliceof.Object[geosearch.Place], error) {

	const location = "service.Geosearch.Geosearch"

	providerFunc := service.getProvider(session, referer)

	result, err := providerFunc(query)

	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to retrieve search results")
	}

	return result, nil
}

// getProvider returns the provider configured for this domain.
// If none is configured, then the "free" OpenStreetMap provider is used.
func (service Geosearch) getProvider(session data.Session, referer string) geosearch.GeosearchFunc {

	const location = "service.geosearch.getProvider"

	// Get the provider connction config
	connection := model.NewConnection()

	if err := service.connectionService.LoadActiveByType(session, model.ConnectionTypeGeosearch, &connection); err != nil {
		derp.Report(derp.Wrap(err, location, "Error loading provider connection"))
		return geosearch.Nil()
	}

	switch connection.ProviderID {

	case model.ConnectionProviderGeoapify:
		return geosearch.Geoapify(connection.Data.GetString("apiKey"))

	case model.ConnectionProviderNominatim:
		return geosearch.Nominatim(connection.Data.GetString("serverUrl"), "Emissary (http://emissary.dev) / "+service.hostname, referer)
	}

	return geosearch.Nil()
}
