package geocoder

import (
	"net/url"
	"strings"

	"github.com/benpate/derp"
	"github.com/benpate/geo"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
)

type Maptiler struct {
	apiKey string
}

func NewMaptiler(apiKey string) Maptiler {

	return Maptiler{
		apiKey: apiKey,
	}
}

func (geocoder Maptiler) GeocodeAddress(address string) (point geo.Address, err error) {

	const location = "service.geocoder.Maptiler.GeocodeNetwork"

	// Request IP address location from Maptiler
	endpoint := "https://api.maptiler.com/geocoding/" + url.PathEscape(address) + ".json"
	response := mapof.NewAny()

	txn := remote.Get(endpoint).
		Query("key", geocoder.apiKey).
		Result(&response)

	if err := txn.Send(); err != nil {
		return geo.Address{}, derp.Wrap(err, location, "Error returned by Maptiler")
	}

	// Parse response
	var features sliceof.Object[mapof.Any] = response.GetSliceOfMap("features")

	if features.IsZero() {
		return geo.Address{}, derp.NotFound(location, "Address not found", address)
	}

	return mapMaptilerAddress(features.First()), nil
}

func (geocoder Maptiler) AutocompleteAddress(query string) (sliceof.Object[geo.Address], error) {

	const location = "service.geocoder.Maptiler.AutocompleteAddress"

	// Request IP address location from Maptiler
	endpoint := "https://api.maptiler.com/geocoding/" + url.PathEscape(query) + ".json"
	response := mapof.NewAny()

	txn := remote.Get(endpoint).
		Query("key", geocoder.apiKey).
		Result(&response)

	if err := txn.Send(); err != nil {
		return nil, derp.Wrap(err, location, "Error returned by Maptiler")
	}

	// Map "features" from the result into geo.Address
	features := response.GetSliceOfMap("features")
	features = slice.Filter(features, filterMaptilerAddress)
	result := slice.Map(features, mapMaptilerAddress)
	return result, nil
}

/*
MAPTILER Network lookups are disabled, because they can
only return the location of the SERVER, and not the
location of the USER'S machine.  Sooo close :(
*/

func filterMaptilerAddress(feature mapof.Any) bool {
	return feature.GetString("place_type") == "address"
}

func mapMaptilerAddress(feature mapof.Any) geo.Address {

	point := geo.Point{}

	if err := point.UnmarshalMap(feature.GetMap("geometry")); err != nil {
		derp.Report(derp.Wrap(err, "service.geocoder.mapMaptilerAddress", "Invalid geometry"))
	}

	address := geo.Address{}
	address.Name = first.String(feature.GetString("place_name"), feature.GetString("matching_place_name"))
	address.Longitude = point.Longitude
	address.Latitude = point.Latitude

	address.Street1 = feature.GetString("address") + " " + feature.GetString("text")

	context := feature.GetSliceOfMap("context")
	for _, item := range context {

		switch item.GetString("place_designation") {
		case "":
			if id := item.GetString("id"); strings.HasPrefix(id, "postal_code") {
				address.PostalCode = item.GetString("text")
			}
		case "city":
			address.Locality = item.GetString("text")
		case "state", "province":
			address.Region = item.GetString("text")
		case "country":
			address.Country = item.GetString("text")
		}
	}

	return address
}
