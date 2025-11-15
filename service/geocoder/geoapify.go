package geocoder

import (
	"strings"

	"github.com/benpate/derp"
	"github.com/benpate/geo"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
)

type Geoapify struct {
	apiKey string
}

func NewGeoapify(apiKey string) Geoapify {

	return Geoapify{
		apiKey: apiKey,
	}
}

func (geocoder Geoapify) GeocodeAddress(address string) (geo.Address, error) {

	const location = "service.geocoder.Geoapify.GeocodeNetwork"

	// Request IP address location from Geoapify
	response := mapof.NewAny()

	txn := remote.Get("https://api.geoapify.com/v1/geocode/search").
		Query("text", address).
		Query("apiKey", geocoder.apiKey).
		Result(&response)

	if err := txn.Send(); err != nil {
		return geo.Address{}, derp.Wrap(err, location, "Error returned by Geoapify")
	}

	// Find the Properties for the first Feature in the response
	var features sliceof.MapOfAny = response.GetSliceOfMap("features")
	feature := features.First()

	// Let's make us an address...
	result := mapGeoapifyAddress(feature)
	return result, nil
}

func (geocoder Geoapify) GeocodeTimezone(address *geo.Address) error {

	const location = "service.geocoder.Geoapify.GeocodeTimezone"

	result, err := geocoder.GeocodeAddress(address.Formatted)

	if err != nil {
		return derp.Wrap(err, location, "Unable to retrieve timezone information")
	}

	address.Timezone = result.Timezone
	return nil
}

func (geocoder Geoapify) GeocodeNetwork(ip string) (point geo.Point, err error) {

	// Request IP address location from Geoapify
	response := mapof.NewAny()
	txn := remote.Get("https://api.geoapify.com/v1/ipinfo").
		Query("apiKey", geocoder.apiKey).
		Query("ip", ip).
		Result(&response)

	if err := txn.Send(); err != nil {
		return geo.Point{}, derp.Wrap(err, "service.geocoder.Geoapify.GeocodeNetwork", "Error calling IPAPICOM")
	}

	// Parse response
	location := response.GetMap("location")
	latitude := location.GetFloat("latitude")
	longitude := location.GetFloat("longitude")

	// Return result
	return geo.NewPoint(longitude, latitude), nil
}

func (geocoder Geoapify) AutocompleteAddress(query string, bias geo.Point) (sliceof.Object[geo.Address], error) {

	const location = "service.geocoder.Geoapify.AutocompleteAddress"

	response := mapof.NewAny()

	txn := remote.Get("https://api.geoapify.com/v1/geocode/autocomplete").
		Query("text", query).
		Query("apiKey", geocoder.apiKey).
		Result(&response)

	if bias.NotZero() {
		txn.Query("bias", "proximity:"+bias.LonLat())
	}

	if err := txn.Send(); err != nil {
		return nil, derp.Wrap(err, location, "Unable to retrieve search results")
	}

	// Map "features" from the result into geo.Address
	features := response.GetSliceOfMap("features")
	result := slice.Map(features, mapGeoapifyAddress)
	return result, nil
}

func mapGeoapifyAddress(feature mapof.Any) geo.Address {

	properties := feature.GetMap("properties")

	return geo.Address{
		Name:       first.String(properties.GetString("name"), properties.GetString("formatted")),
		Formatted:  properties.GetString("formatted"),
		Street1:    strings.TrimSpace(properties.GetString("housenumber") + " " + properties.GetString("street")),
		Locality:   properties.GetString("city"),
		Region:     properties.GetString("state"),
		PostalCode: properties.GetString("postcode"),
		Country:    properties.GetString("country"),
		PlusCode:   properties.GetString("plus_code"),
		Longitude:  properties.GetFloat("lon"),
		Latitude:   properties.GetFloat("lat"),
		Timezone:   properties.GetMap("timezone").GetString("name"),
	}
}
