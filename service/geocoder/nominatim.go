package geocoder

import (
	"github.com/benpate/derp"
	"github.com/benpate/geo"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
)

// https://nominatim.org
type Nominatim struct {
	searchURL string
	apiKey    string
	userAgent string
	referer   string
}

func NewNominatim(searchURL string, apiKey string, userAgent string, referer string) Nominatim {

	// Default value uses public server.
	if searchURL == "" {
		searchURL = "https://nominatim.openstreetmap.org"
	}

	return Nominatim{
		searchURL: searchURL,
		apiKey:    apiKey,
		userAgent: userAgent,
		referer:   referer,
	}
}

func (geocoder Nominatim) GeocodeAddress(address string) (point geo.Address, err error) {

	const location = "service.ggeocoder.Nominatim.AutocompleteAddress"

	// Send the request to the Nominatim server
	response := make(sliceof.Object[mapof.Any], 0)
	txn := remote.Get(geocoder.searchURL+"/search").
		UserAgent(geocoder.userAgent).
		Header("Referer", geocoder.referer).
		Query("q", address).
		Query("format", "jsonv2").
		Result(&response)

	if err := txn.Send(); err != nil {
		return geo.Address{}, derp.Wrap(err, location, "Unable to retrieve search results")
	}

	if response.IsZero() {
		return geo.Address{}, derp.NotFound(location, "Address not found", address)
	}

	place := response.First()

	// Map addresses into interface format and return
	return geo.Address{
		Formatted: address,
		Longitude: place.GetFloat("lon"),
		Latitude:  place.GetFloat("lat"),
	}, nil
}

func (geocoder Nominatim) AutocompleteAddress(query string) (sliceof.Object[geo.Address], error) {

	const location = "service.ggeocoder.Nominatim.AutocompleteAddress"

	// Send the request to the Nominatim server
	response := make(sliceof.Object[mapof.Any], 0)
	txn := remote.Get(geocoder.searchURL+"/search").
		UserAgent(geocoder.userAgent).
		Header("Referer", geocoder.referer).
		Query("q", query).
		Query("format", "jsonv2").
		Result(&response)

	if err := txn.Send(); err != nil {
		return nil, derp.Wrap(err, location, "Unable to retrieve search results")
	}

	// Map addresses into interface format and return
	result := slice.Map(response, mapNominatimAddress)
	return result, nil
}

func mapNominatimAddress(place mapof.Any) geo.Address {
	return geo.Address{
		Name:      place.GetString("display_name"),
		Latitude:  place.GetFloat("lat"),
		Longitude: place.GetFloat("lon"),
	}
}
