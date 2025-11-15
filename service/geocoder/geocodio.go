package geocoder

import (
	"github.com/benpate/derp"
	"github.com/benpate/geo"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
)

type Geocodio struct {
	apiKey string
}

func NewGeocodio(apiKey string) Geocodio {
	return Geocodio{
		apiKey: apiKey,
	}
}

func (service Geocodio) GeocodeAddress(address string) (geo.Address, error) {

	const location = "service.geocoder.Geocodio.GeocodeTimezone"

	// Connect to the Geocodio API server
	response := mapof.NewAny()
	txn := remote.Get("https://api.geocod.io/v1.9/geocode").
		Query("q", address).
		Query("fields", "timezone").
		Query("api_key", service.apiKey).
		Result(&response)

	if err := txn.Send(); err != nil {
		return geo.Address{}, derp.Wrap(err, location, "Unable to connect to Geocodio API server")
	}

	// Parse API results
	var apiResults sliceof.MapOfAny = response.GetSliceOfMap("results")

	if apiResults.IsZero() {
		return geo.Address{}, derp.NotFound(location, "No results returned by Geocodio API server")
	}

	apiResult := apiResults.First()
	result := geo.Address{}

	// Populate address line 1, address line 2
	var address_lines sliceof.String = apiResult.GetSliceOfString("address_lines")
	result.Street1 = address_lines.At(0)
	result.Street2 = address_lines.At(1)

	// Populate city, state, country
	address_components := apiResult.GetMap("address_components")
	result.Locality = address_components.GetString("city")
	result.Region = address_components.GetString("state")
	result.PostalCode = address_components.GetString("zip")
	result.Country = address_components.GetString("country")

	address_location := apiResult.GetMap("location")
	result.Longitude = address_location.GetFloat("lng")
	result.Latitude = address_location.GetFloat("lat")

	fields := apiResult.GetMap("fields")
	timezone := fields.GetMap("timezone")
	result.Timezone = timezone.GetString("name")

	return result, nil
}

func (service Geocodio) GeocodeTimezone(address *geo.Address) error {

	const location = "service.geocoder.Geocodio.GeocodeTimezone"

	// Connect to the Geocodio API server
	response := mapof.NewAny()
	txn := remote.Get("https://api.geocod.io/v1.9/geocode").
		Query("q", address.Formatted).
		Query("fields", "timezone").
		Query("api_key", service.apiKey).
		Result(&response)

	if err := txn.Send(); err != nil {
		return derp.Wrap(err, location, "Unable to connect to Geocodio API server")
	}

	// Parse results
	var apiResults sliceof.MapOfAny = response.GetSliceOfMap("results")

	if apiResults.IsZero() {
		return derp.NotFound(location, "No results returned by Geocodio API server")
	}

	// Extract the Timezone from the response
	apiResult := apiResults.First()
	fields := apiResult.GetMap("fields")
	timezone := fields.GetMap("timezone")
	address.Timezone = timezone.GetString("name")

	// Yup.
	return nil
}
