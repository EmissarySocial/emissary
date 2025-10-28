package model

import (
	"github.com/benpate/rosetta/schema"
)

func PlaceSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"name":        schema.String{},
			"fullAddress": schema.String{},
			"street1":     schema.String{},
			"street2":     schema.String{},
			"locality":    schema.String{},
			"region":      schema.String{},
			"postalCode":  schema.String{},
			"country":     schema.String{},
			"latitude":    schema.Number{},
			"longitude":   schema.Number{},
			"radius":      schema.Number{},
			"units":       schema.String{},
		},
	}
}

/******************************************
 * Getters
 ******************************************/

func (place *Place) GetPointer(name string) (any, bool) {

	switch name {

	case "name":
		return place.Name, true

	case "fullAddress":
		return place.FullAddress, true

	case "street1":
		return place.Street1, true

	case "street2":
		return place.Street2, true

	case "locality":
		return place.Locality, true

	case "region":
		return place.Region, true

	case "postalCode":
		return place.PostalCode, true

	case "country":
		return place.Country, true

	case "latitude":
		return place.Latitude, true

	case "longitude":
		return place.Longitude, true

	case "radius":
		return place.Radius, true

	case "units":
		return place.Units, true
	}

	return nil, false
}

func (place *Place) GetStringOK(name string) (string, bool) {

	switch name {

	case "name":
		return place.Name, true

	case "fullAddress":
		return place.FullAddress, true

	case "street1":
		return place.Street1, true

	case "street2":
		return place.Street2, true

	case "locality":
		return place.Locality, true

	case "region":
		return place.Region, true

	case "postalCode":
		return place.PostalCode, true

	case "country":
		return place.Country, true

	case "units":
		return place.Units, true
	}

	return "", false
}

func (place *Place) GetFloat(name string) (float64, bool) {

	switch name {

	case "latitude":
		return place.Latitude, true

	case "longitude":
		return place.Longitude, true

	case "radius":
		return place.Radius, true
	}

	return 0, false
}

/******************************************
 * Setters
 ******************************************/

func (place *Place) SetString(name string, value string) bool {

	switch name {

	case "name":
		place.Name = value
		return true

	case "fullAddress":
		if value != place.FullAddress {
			place.FullAddress = value
			place.Reset()
		}
		return true
	}

	return false
}

func (place *Place) SetFloat(name string, value float64) bool {

	switch name {

	case "latitude":
		place.Latitude = value
		return true

	case "longitude":
		place.Longitude = value
		return true

	case "radius":
		place.Radius = value
		return true
	}

	return false
}
