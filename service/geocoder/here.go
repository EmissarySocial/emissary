package geocoder

import (
	"strings"

	"github.com/benpate/derp"
	"github.com/benpate/geo"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
)

// Consider: https://www.here.com

type Here struct {
	apiID  string
	apiKey string
}

func NewHere(apiID string, apiKey string) Here {
	return Here{
		apiID:  apiID,
		apiKey: apiKey,
	}
}

func (geocoder Here) GeocodeAddress(q string) (geo.Address, error) {

	const location = "service.geocoder.Geoapify.GeocodeNetwork"

	// Request IP address location from Geoapify
	response := mapof.NewAny()

	txn := remote.Get("https://geocode.search.hereapi.com/v1/geocode").
		Query("q", q).
		Query("show", "tz").
		Query("apiKey", geocoder.apiKey).
		Result(&response)

	if err := txn.Send(); err != nil {
		return geo.Address{}, derp.Wrap(err, location, "Error returned by Here.com API")
	}

	var items sliceof.MapOfAny = response.GetSliceOfMap("items")

	if items.IsZero() {
		return geo.Address{}, derp.NotFound(location, "Location not found by Here.com API")
	}

	return mapHereAddress(items.First()), nil
}

func (geocoder Here) GeocodeTimezone(address *geo.Address) error {

	const location = "service.geocoder.Here.GeocodeNetwork"

	// Request IP address location from Here.com
	response := mapof.NewAny()

	txn := remote.Get("https://geocode.search.hereapi.com/v1/geocode").
		Query("q", address.Formatted).
		Query("show", "tz").
		Query("apiKey", geocoder.apiKey).
		Result(&response)

	if err := txn.Send(); err != nil {
		derp.Report(err)
		return derp.Wrap(err, location, "Error returned by Here.com API")
	}

	var items sliceof.MapOfAny = response.GetSliceOfMap("items")

	if items.IsZero() {
		return derp.NotFound(location, "Location not found by Here.com API")
	}

	item := items.First()
	timezone := item.GetMap("timeZone")
	address.Timezone = timezone.GetString("name")

	return nil
}

func (geocoder Here) AutocompleteAddress(query string, bias geo.Point) (sliceof.Object[geo.Address], error) {

	const location = "service.geocoder.Here.AutocompleteAddress"

	response := mapof.NewAny()

	txn := remote.Get("https://autocomplete.search.hereapi.com/v1/autocomplete").
		Query("q", query).
		Query("apiKey", geocoder.apiKey).
		Result(&response)

	// If a non-zero bias point is present, then include that in the search query
	if bias.NotZero() {
		txn.Query("at", bias.LatLon())
	}

	if err := txn.Send(); err != nil {
		return nil, derp.Wrap(err, location, "Unable to retrieve search results")
	}

	// Map "features" from the result into geo.Address
	features := response.GetSliceOfMap("items")
	result := slice.Map(features, mapHereAddress)
	return result, nil
}

func mapHereAddress(item mapof.Any) geo.Address {

	address := item.GetMap("address")

	result := geo.Address{}
	result.Name = address.GetString("label")
	result.Formatted = address.GetString("label")
	result.Street1, _, _ = strings.Cut(result.Formatted, ",")
	result.Locality = address.GetString("city")
	result.Region = address.GetString("state")
	result.PostalCode = address.GetString("postalCode")
	result.Country = address.GetString("countryName")

	timezone := item.GetMap("timeZone")
	result.Timezone = timezone.GetString("name")

	position := item.GetMap("position")
	result.Longitude = position.GetFloat("lng")
	result.Latitude = position.GetFloat("lat")
	return result
}
