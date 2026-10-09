package geocoder

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

/******************************************
 * Response mapping tests
 *
 * Every geocoder converts a provider's JSON into a geo.Address with a
 * small pure function. A wrong key name there produces a silently empty
 * field rather than an error, so these pin the key names.
 ******************************************/

// hereResult is one result from the HERE geocoding API, trimmed to the keys the mapper reads.
func hereResult() mapof.Any {
	return mapof.Any{
		"address": mapof.Any{
			"label":       "3317 E Colfax Ave, Denver, CO 80206, United States",
			"city":        "Denver",
			"state":       "Colorado",
			"postalCode":  "80206",
			"countryName": "United States",
			"countryCode": "USA",
		},
		"position": mapof.Any{"lat": 39.7392, "lng": -104.9903},
		"timeZone": mapof.Any{"name": "America/Denver"},
	}
}

// TestMapHereAddress pins every field the HERE mapper reads.
func TestMapHereAddress(t *testing.T) {

	address := mapHereAddress(hereResult())

	require.Equal(t, "3317 E Colfax Ave, Denver, CO 80206, United States", address.Name)
	require.Equal(t, "3317 E Colfax Ave, Denver, CO 80206, United States", address.Formatted)
	require.Equal(t, "3317 E Colfax Ave", address.Street1, "Street1 is the label up to the first comma")
	require.Equal(t, "Denver", address.Locality)
	require.Equal(t, "Colorado", address.Region)
	require.Equal(t, "80206", address.PostalCode)
	require.Equal(t, "United States", address.Country)
	require.Equal(t, "America/Denver", address.Timezone)
	require.Equal(t, -104.9903, address.Longitude)
	require.Equal(t, 39.7392, address.Latitude)
}

// TestMapHereAddress_LabelWithoutComma confirms that a single-part label becomes
// Street1 whole, because strings.Cut returns the input when the separator is absent.
func TestMapHereAddress_LabelWithoutComma(t *testing.T) {

	result := hereResult()
	result["address"] = mapof.Any{"label": "Denver"}

	require.Equal(t, "Denver", mapHereAddress(result).Street1)
}

// TestMapHereAddress_Empty confirms that a result with no recognized keys maps to a
// zero Address instead of panicking on a missing sub-document.
func TestMapHereAddress_Empty(t *testing.T) {

	require.True(t, mapHereAddress(mapof.NewAny()).IsZero())
	require.True(t, mapHereAddress(nil).IsZero())
}

// TestMapNominatimAddress pins the Nominatim mapper, whose coordinates arrive as
// STRINGS in the jsonv2 format rather than as numbers.
func TestMapNominatimAddress(t *testing.T) {

	address := mapNominatimAddress(mapof.Any{
		"display_name": "3317, East Colfax Avenue, Denver, Colorado, 80206, United States",
		"lat":          "39.7392358",
		"lon":          "-104.9847034",
	})

	require.Equal(t, "3317, East Colfax Avenue, Denver, Colorado, 80206, United States", address.Name)
	require.InDelta(t, 39.7392358, address.Latitude, 0.0000001, "string coordinates must convert")
	require.InDelta(t, -104.9847034, address.Longitude, 0.0000001)
}

// TestMapNominatimAddress_NumericCoordinates confirms the mapper also accepts
// coordinates that arrive as real JSON numbers.
func TestMapNominatimAddress_NumericCoordinates(t *testing.T) {

	address := mapNominatimAddress(mapof.Any{"lat": 39.7392, "lon": -104.9903})

	require.Equal(t, 39.7392, address.Latitude)
	require.Equal(t, -104.9903, address.Longitude)
}

// TestMapNominatimAddress_Empty confirms an unrecognized result maps to a zero Address.
func TestMapNominatimAddress_Empty(t *testing.T) {
	require.True(t, mapNominatimAddress(mapof.NewAny()).IsZero())
}

// googleResult is one result from the Google geocoding API, trimmed to the keys the mapper reads.
func googleResult() mapof.Any {
	return mapof.Any{
		"formatted_address": "3317 E Colfax Ave, Denver, CO 80206, USA",
		"geometry": mapof.Any{
			"location": mapof.Any{"lat": 39.7392, "lng": -104.9903},
		},
		"address_components": []any{
			mapof.Any{"long_name": "3317", "types": []any{"street_number"}},
			mapof.Any{"long_name": "East Colfax Avenue", "types": []any{"route"}},
			mapof.Any{"long_name": "Denver", "types": []any{"locality", "political"}},
			mapof.Any{"long_name": "Colorado", "types": []any{"administrative_area_level_1", "political"}},
			mapof.Any{"long_name": "United States", "types": []any{"country", "political"}},
			mapof.Any{"long_name": "80206", "types": []any{"postal_code"}},
		},
	}
}

// TestMapGoogleSearchResult pins the fields the Google mapper populates correctly.
func TestMapGoogleSearchResult(t *testing.T) {

	address := mapGoogleSearchResult(googleResult())

	require.Equal(t, "3317 E Colfax Ave, Denver, CO 80206, USA", address.Formatted)
	require.Equal(t, "3317 East Colfax Avenue", address.Street1, "street_number then route")
	require.Equal(t, "Denver", address.Locality)
	require.Equal(t, "80206", address.PostalCode)
	require.Equal(t, "United States", address.Country)
	require.Equal(t, -104.9903, address.Longitude)
	require.Equal(t, 39.7392, address.Latitude)
}

// TestMapGoogleSearchResult_RegionIsNeverPopulated documents a live defect: the switch
// matches "administrative_level_1", but Google sends "administrative_area_level_1".
func TestMapGoogleSearchResult_RegionIsNeverPopulated(t *testing.T) {

	address := mapGoogleSearchResult(googleResult())

	// Asserting the BUG, not the intent. Delete this test when the key is corrected.
	require.Equal(t, "", address.Region, "Region is silently dropped -- the mapper looks for the wrong key")
}

// TestMapGoogleSearchResult_RouteBeforeNumber confirms Street1 assembles correctly when
// the route component arrives before the street number.
func TestMapGoogleSearchResult_RouteBeforeNumber(t *testing.T) {

	result := googleResult()
	result["address_components"] = []any{
		mapof.Any{"long_name": "East Colfax Avenue", "types": []any{"route"}},
		mapof.Any{"long_name": "3317", "types": []any{"street_number"}},
	}

	require.Equal(t, "3317 East Colfax Avenue", mapGoogleSearchResult(result).Street1)
}

// TestMapGoogleSearchResult_Empty confirms a result with no components maps cleanly.
func TestMapGoogleSearchResult_Empty(t *testing.T) {

	address := mapGoogleSearchResult(mapof.NewAny())

	require.True(t, address.IsZero())
	require.Equal(t, "", address.Street1)
}

// TestMapGooglePlaceSuggestion pins the autocomplete mapper, which carries a label only.
func TestMapGooglePlaceSuggestion(t *testing.T) {

	address := mapGooglePlaceSuggestion(mapof.Any{
		"placePrediction": mapof.Any{
			"text": mapof.Any{"text": "3317 E Colfax Ave, Denver, CO, USA"},
		},
	})

	require.Equal(t, "3317 E Colfax Ave, Denver, CO, USA", address.Name)
	require.Equal(t, "3317 E Colfax Ave, Denver, CO, USA", address.Formatted)
	require.False(t, address.HasGeocode(), "autocomplete suggestions carry no coordinates")
}

// TestMapGooglePlaceSuggestion_Empty confirms a malformed suggestion maps to a zero Address.
func TestMapGooglePlaceSuggestion_Empty(t *testing.T) {
	require.True(t, mapGooglePlaceSuggestion(mapof.NewAny()).IsZero())
}

// geoapifyFeature is one feature from the Geoapify API, trimmed to the keys the mapper reads.
func geoapifyFeature() mapof.Any {
	return mapof.Any{
		"properties": mapof.Any{
			"name":        "Bluebird Theater",
			"formatted":   "Bluebird Theater, 3317 E Colfax Ave, Denver, CO 80206",
			"housenumber": "3317",
			"street":      "E Colfax Ave",
			"city":        "Denver",
			"state":       "Colorado",
			"postcode":    "80206",
			"country":     "United States",
			"plus_code":   "85FPQXX5+9F",
			"lon":         -104.9903,
			"lat":         39.7392,
			"timezone":    mapof.Any{"name": "America/Denver"},
		},
	}
}

// TestMapGeoapifyAddress pins every field the Geoapify mapper reads.
func TestMapGeoapifyAddress(t *testing.T) {

	address := mapGeoapifyAddress(geoapifyFeature())

	require.Equal(t, "Bluebird Theater", address.Name)
	require.Equal(t, "Bluebird Theater, 3317 E Colfax Ave, Denver, CO 80206", address.Formatted)
	require.Equal(t, "3317 E Colfax Ave", address.Street1)
	require.Equal(t, "Denver", address.Locality)
	require.Equal(t, "Colorado", address.Region)
	require.Equal(t, "80206", address.PostalCode)
	require.Equal(t, "United States", address.Country)
	require.Equal(t, "85FPQXX5+9F", address.PlusCode)
	require.Equal(t, "America/Denver", address.Timezone)
	require.Equal(t, -104.9903, address.Longitude)
	require.Equal(t, 39.7392, address.Latitude)
}

// TestMapGeoapifyAddress_NameFallsBackToFormatted confirms the first.String fallback
// when a feature carries no name of its own.
func TestMapGeoapifyAddress_NameFallsBackToFormatted(t *testing.T) {

	feature := geoapifyFeature()
	properties := feature.GetMap("properties")
	delete(properties, "name")
	feature["properties"] = properties

	address := mapGeoapifyAddress(feature)
	require.Equal(t, address.Formatted, address.Name)
}

// TestMapGeoapifyAddress_StreetTrimsWhitespace confirms Street1 has no stray space when
// only one of housenumber and street is present.
func TestMapGeoapifyAddress_StreetTrimsWhitespace(t *testing.T) {

	onlyStreet := mapof.Any{"properties": mapof.Any{"street": "E Colfax Ave"}}
	require.Equal(t, "E Colfax Ave", mapGeoapifyAddress(onlyStreet).Street1)

	onlyNumber := mapof.Any{"properties": mapof.Any{"housenumber": "3317"}}
	require.Equal(t, "3317", mapGeoapifyAddress(onlyNumber).Street1)
}

// TestMapGeoapifyAddress_Empty confirms an unrecognized feature maps to a zero Address.
func TestMapGeoapifyAddress_Empty(t *testing.T) {
	require.True(t, mapGeoapifyAddress(mapof.NewAny()).IsZero())
}

// maptilerFeature is one feature from the MapTiler API, trimmed to the keys the mapper reads.
func maptilerFeature() mapof.Any {
	return mapof.Any{
		"place_name": "3317 E Colfax Ave, Denver, Colorado 80206, United States",
		"text":       "E Colfax Ave",
		"address":    "3317",
		"geometry": mapof.Any{
			"type":        "Point",
			"coordinates": []any{-104.9903, 39.7392},
		},
		"context": []any{
			mapof.Any{"id": "postal_code.5678", "text": "80206"},
			mapof.Any{"id": "place.1234", "place_designation": "city", "text": "Denver"},
			mapof.Any{"id": "region.9012", "place_designation": "state", "text": "Colorado"},
			mapof.Any{"id": "country.3456", "place_designation": "country", "text": "United States"},
		},
	}
}

// TestMapMaptilerAddress pins every field the MapTiler mapper reads, including the
// context list that carries locality, region, country and postal code.
func TestMapMaptilerAddress(t *testing.T) {

	address := mapMaptilerAddress(maptilerFeature())

	require.Equal(t, "3317 E Colfax Ave, Denver, Colorado 80206, United States", address.Name)
	require.Equal(t, "3317 E Colfax Ave", address.Street1)
	require.Equal(t, "80206", address.PostalCode, "postal code is recognized by its id prefix")
	require.Equal(t, "Denver", address.Locality)
	require.Equal(t, "Colorado", address.Region)
	require.Equal(t, "United States", address.Country)
	require.Equal(t, -104.9903, address.Longitude)
	require.Equal(t, 39.7392, address.Latitude)
}

// TestMapMaptilerAddress_ProvinceIsARegion confirms that "province" maps to Region
// alongside "state", which is how MapTiler labels non-US subdivisions.
func TestMapMaptilerAddress_ProvinceIsARegion(t *testing.T) {

	feature := maptilerFeature()
	feature["context"] = []any{
		mapof.Any{"id": "region.1", "place_designation": "province", "text": "Ontario"},
	}

	require.Equal(t, "Ontario", mapMaptilerAddress(feature).Region)
}

// TestMapMaptilerAddress_NameFallsBackToMatchingPlaceName confirms the first.String
// fallback when a feature carries no place_name.
func TestMapMaptilerAddress_NameFallsBackToMatchingPlaceName(t *testing.T) {

	feature := maptilerFeature()
	delete(feature, "place_name")
	feature["matching_place_name"] = "Bluebird Theater"

	require.Equal(t, "Bluebird Theater", mapMaptilerAddress(feature).Name)
}

// TestMapMaptilerAddress_InvalidGeometry confirms that unparseable geometry is reported
// and skipped, leaving the rest of the address intact rather than failing the whole result.
func TestMapMaptilerAddress_InvalidGeometry(t *testing.T) {

	feature := maptilerFeature()
	feature["geometry"] = mapof.Any{"type": "Polygon", "coordinates": []any{1.0, 2.0}}

	address := mapMaptilerAddress(feature)

	require.False(t, address.HasGeocode(), "bad geometry yields no coordinates")
	require.Equal(t, "Denver", address.Locality, "...but the rest of the address survives")
}

// TestMapMaptilerAddress_Empty documents a live defect: an unrecognized feature builds
// Street1 by concatenating two empty strings around a space, producing a lone " ".
func TestMapMaptilerAddress_Empty(t *testing.T) {

	address := mapMaptilerAddress(mapof.NewAny())

	require.False(t, address.HasGeocode())

	// Asserting the BUG, not the intent. A lone space is not an empty Street1, so an
	// empty result reports HasAddress() == TRUE and renders a blank address line.
	require.Equal(t, " ", address.Street1)
	require.True(t, address.HasAddress(), "a result with no address claims to have one")
}
