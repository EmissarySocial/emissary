package geocoder

import (
	"github.com/benpate/derp"
	"github.com/benpate/geo"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
)

type GoogleMaps struct {
	apiKey string
}

func NewGoogleMaps(apiKey string) GoogleMaps {

	return GoogleMaps{
		apiKey: apiKey,
	}
}

func (geocoder GoogleMaps) GeocodeAddress(query string) (geo.Address, error) {

	const location = "service.geocoder.GoogleMaps.GeocodeAddress"

	// Connect to Google to Retrieve Address Information
	response := mapof.NewAny()
	txn := remote.Get("https://maps.googleapis.com/maps/api/geocode/json").
		Query("key", geocoder.apiKey).
		Query("address", query).
		Result(&response)

	if err := txn.Send(); err != nil {
		return geo.Address{}, derp.Wrap(err, location, "Unable to load results from Google Places API", query)
	}

	// Parse the Results
	var results sliceof.Object[mapof.Any] = response.GetSliceOfMap("results")

	if results.IsEmpty() {
		return geo.Address{}, derp.NotFound(location, "Address not found")
	}

	// Convert Google's awful format into something we can use
	address := mapGoogleSearchResult(results.First())

	// Success, if you call it that...
	return address, nil
}

func (geocoder GoogleMaps) AutocompleteAddress(query string) (sliceof.Object[geo.Address], error) {

	const location = "service.geocoder.GoogleMaps.AutocompleteAddress"

	body := mapof.String{
		"textQuery": query,
	}

	response := mapof.NewAny()

	txn := remote.Post("https://places.googleapis.com/v1/places:searchText").
		JSON(body).
		ContentType("application/json").
		Header("X-Goog-Api-Key", geocoder.apiKey).
		Header("X-Goog-Fieldmask", "places.formattedAddress,places.location,places.types").
		Result(&response)

	if err := txn.Send(); err != nil {
		return nil, derp.Wrap(err, location, "Unable to load results from Google Places API", query)
	}

	places := response.GetSliceOfMap("places")
	addresses := slice.Map(places, mapGoogleAddress)
	return addresses, nil
}

func mapGoogleAddress(place mapof.Any) geo.Address {

	location := place.GetMap("location")
	longitude := location.GetFloat("longitude")
	latitude := location.GetFloat("latitude")

	return geo.Address{
		Name:      place.GetString("formattedAddress"),
		Longitude: longitude,
		Latitude:  latitude,
	}
}

func mapGoogleSearchResult(result mapof.Any) geo.Address {

	address := geo.NewAddress()
	address.Formatted = result.GetString("formatted_address")

	// Collect Longitude/Latutude
	geometry := result.GetMap("geometry")
	location := geometry.GetMap("location")
	address.Longitude = location.GetFloat("lng")
	address.Latitude = location.GetFloat("lat")

	// Collect Address Components
	address_components := result.GetSliceOfMap("address_components")
	for _, component := range address_components {

		for _, componentType := range component.GetSliceOfString("types") {

			switch componentType {

			case "street_number":
				if address.Street1 == "" {
					address.Street1 = component.GetString("long_name")
				} else {
					address.Street1 = component.GetString("long_name") + " " + address.Street1
				}

			case "route":
				if address.Street1 == "" {
					address.Street1 = component.GetString("long_name")
				} else {
					address.Street1 = address.Street1 + " " + component.GetString("long_name")
				}

			case "locality":
				address.Locality = component.GetString("long_name")

			case "administrative_level_1":
				address.Region = component.GetString("long_name")

			case "country":
				address.Country = component.GetString("long_name")

			case "postal_code":
				address.PostalCode = component.GetString("long_name")
			}
		}
	}

	return address
}
