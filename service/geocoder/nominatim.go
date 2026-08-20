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

// NewNominatim returns a Nominatim geocoder
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

// GeocodeAddress resolves a free-text address into a geo.Address. Implements the AddressGeocoder interface.
func (geocoder Nominatim) GeocodeAddress(query string) (point geo.Address, err error) {

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
		return geo.Address{}, derp.Wrap(err, location, "Retrieving search results")
	}

	if response.IsZero() {
		return geo.Address{}, derp.NotFound(location, "Address not found", query)
	}

	place := response.First()

	// Map address into interface format and return
	return mapNominatimAddress(place), nil
}

// AutocompleteAddress returns address suggestions for a partial query, biased toward the provided point. Implements the AddressAutocompleter interface.
func (geocoder Nominatim) AutocompleteAddress(query string, bias geo.Point) (sliceof.Object[geo.Address], error) {

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
		return nil, derp.Wrap(err, location, "Retrieving search results")
	}

	// Map addresses into interface format and return
	result := slice.Map(response, mapNominatimAddress)
	return result, nil
}

// mapNominatimAddress converts one Nominatim API result into a geo.Address
func mapNominatimAddress(place mapof.Any) geo.Address {
	return geo.Address{
		Name:      place.GetString("display_name"),
		Latitude:  place.GetFloat("lat"),
		Longitude: place.GetFloat("lon"),
	}
}
