package geocoder

import (
	"strconv"
	"time"

	"github.com/benpate/derp"
	"github.com/benpate/geo"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
)

// GoogleMaps geocodes addresses and timezones using the Google Maps Platform API
type GoogleMaps struct {
	apiKey string
}

// NewGoogleMaps returns a GoogleMaps geocoder
func NewGoogleMaps(apiKey string) GoogleMaps {

	return GoogleMaps{
		apiKey: apiKey,
	}
}

// GeocodeAddress resolves a free-text address into a geo.Address. Implements the AddressGeocoder interface.
func (geocoder GoogleMaps) GeocodeAddress(query string) (geo.Address, error) {

	const location = "service.geocoder.GoogleMaps.GeocodeAddress"

	// Connect to Google to Retrieve Address Information
	response := mapof.NewAny()
	txn := remote.Get("https://maps.googleapis.com/maps/api/geocode/json").
		Query("key", geocoder.apiKey).
		Query("address", query).
		Result(&response)

	if err := txn.Send(); err != nil {
		return geo.Address{}, derp.Wrap(err, location, "Loading results from Google Places API", query)
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

// GeocodeTimezone fills in the timezone of the provided address. Implements the TimezoneGeocoder interface.
func (geocoder GoogleMaps) GeocodeTimezone(address *geo.Address) error {

	const location = "service.geocoder.GoogleMaps.GeocodeTimestamp"

	latLong := address.GetString("latitude") + "," + address.GetString("longitude")
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	// Connect to Google to Retrieve Address Information
	response := mapof.NewAny()
	txn := remote.Get("https://maps.googleapis.com/maps/api/timezone/json").
		Query("location", latLong).
		Query("timestamp", timestamp).
		Query("key", geocoder.apiKey).
		Result(&response)

	if err := txn.Send(); err != nil {
		return derp.Wrap(err, location, "Loading results from Google Timezone API", address)
	}

	address.Timezone = response.GetString("timeZoneId")
	return nil
}

// AutocompleteAddress returns address suggestions for a partial query, biased toward the provided point. Implements the AddressAutocompleter interface.
func (geocoder GoogleMaps) AutocompleteAddress(query string, bias geo.Point) (sliceof.Object[geo.Address], error) {

	const location = "service.geocoder.GoogleMaps.AutocompleteAddress"

	body := mapof.Any{
		"input": query,
	}

	// Add location bias (if present)
	if bias.NotZero() {
		body["locationBias"] = mapof.Any{
			"circle": mapof.Any{
				"center": mapof.Any{
					"longitude": bias.Longitude,
					"latitude":  bias.Latitude,
				},
				"radius": 100.0,
			},
		}
	}

	response := mapof.NewAny()

	txn := remote.Post("https://places.googleapis.com/v1/places:autocomplete").
		JSON(body).
		ContentType("application/json").
		Header("X-Goog-Api-Key", geocoder.apiKey).
		Header("X-Goog-Fieldmask", "suggestions.placePrediction.text.text").
		Result(&response)

	if err := txn.Send(); err != nil {
		return nil, derp.Wrap(err, location, "Loading results from Google Places API", query)
	}

	suggestions := response.GetSliceOfMap("suggestions")
	addresses := slice.Map(suggestions, mapGooglePlaceSuggestion)
	return addresses, nil
}

// mapGooglePlaceSuggestion converts one Google Places autocomplete suggestion into a geo.Address
func mapGooglePlaceSuggestion(suggestion mapof.Any) geo.Address {

	placePrediction := suggestion.GetMap("placePrediction")
	text := placePrediction.GetMap("text")

	return geo.Address{
		Name:      text.GetString("text"),
		Formatted: text.GetString("text"),
	}
}

// mapGoogleSearchResult converts one Google geocoding result into a geo.Address
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
