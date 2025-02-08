package model

import (
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
)

// Place represents a physical place on the planet
// It maps to https://www.w3.org/TR/activitystreams-vocabulary/#dfn-place
// and uses https://schema.org/PostalAddress to match Mobilizion
type Place struct {
	Name        string  `json:"name"        bson:"name"`        // Human-readable name of the place
	FullAddress string  `json:"fullAddress" bson:"fullAddress"` // Full, unparsed address of the place
	Street1     string  `json:"street1"     bson:"street1"`     // Street address line 1 of the place
	Street2     string  `json:"street2"     bson:"street2"`     // Street address line 2 of the place
	Locality    string  `json:"locality"    bson:"locality"`    // City or town of the place
	Region      string  `json:"region"      bson:"region"`      // State or province of the place
	PostalCode  string  `json:"postalCode"  bson:"postalCode"`  // Postal code of the place
	Country     string  `json:"country"     bson:"country"`     // Country of the place
	Latitude    float64 `json:"latitude"    bson:"latitude"`    // Latitude of the place
	Longitude   float64 `json:"longitude"   bson:"longitude"`   // Longitude of the place
	Radius      float64 `json:"radius"      bson:"radius"`      // Radius of the place (in meters)
	Units       string  `json:"units"       bson:"units"`       // Units of measurement for the radius (meters, miles, etc.)
	IsGeocoded  bool    `json:"isGeocoded"  bson:"isGeocoded"`  // True if this place has been geocoded
}

func NewPlace() Place {
	return Place{}
}

// ResetGeocode clears all geocoding information from this Place
func (place *Place) ResetGeocode() {
	place.Street1 = ""
	place.Street2 = ""
	place.Locality = ""
	place.Region = ""
	place.PostalCode = ""
	place.Country = ""
	place.Latitude = 0
	place.Longitude = 0
	place.Radius = 0
	place.Units = ""
	place.IsGeocoded = false
}

// JSONLD returns a JSON-LD representation of this object
func (place Place) JSONLD() mapof.Any {

	result := mapof.Any{
		vocab.PropertyType: vocab.ObjectTypePlace,
	}

	if place.Name != "" {
		result[vocab.PropertyName] = place.Name
	}

	if place.Latitude != 0 && place.Longitude != 0 {
		result[vocab.PropertyLatitude] = place.Latitude
		result[vocab.PropertyLongitude] = place.Longitude
	}

	if place.Radius != 0 {
		result[vocab.PropertyRadius] = place.Radius

		if place.Units != "" {
			result[vocab.PropertyUnits] = place.Units
		}
	}

	if address := place.ParsedAddress(); len(address) > 0 {
		result["address"] = address
	}

	return result
}

func (place Place) ParsedAddress() mapof.String {

	result := mapof.String{}

	if place.Street1 != "" {
		result["streetAddress"] = place.Street1
	}

	if place.Street2 != "" {
		result["streetAddress2"] = place.Street2
	}

	if place.Locality != "" {
		result["addressLocality"] = place.Locality
	}

	if place.Region != "" {
		result["addressRegion"] = place.Region
	}

	if place.PostalCode != "" {
		result["postalCode"] = place.PostalCode
	}

	if place.Country != "" {
		result["addressCountry"] = place.Country
	}

	if len(result) > 0 {
		result["type"] = "PostalAddress"
	}

	return result
}

func (place Place) HasParsedAddress() bool {

	if place.Street1 != "" {
		return true
	}

	if place.Locality != "" {
		return true
	}

	if place.Region != "" {
		return true
	}

	if place.PostalCode != "" {
		return true
	}

	if place.Country != "" {
		return true
	}

	return false
}

func (place Place) IsEmpty() bool {
	return place.FullAddress == ""
}

func (place Place) NotEmpty() bool {
	return !place.IsEmpty()
}

func (place Place) HasGeocode() bool {
	return (place.Latitude != 0) && (place.Longitude != 0)
}
