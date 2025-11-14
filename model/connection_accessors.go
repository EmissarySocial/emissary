package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ConnectionSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"connectionId": schema.String{Format: "objectId"},
			"type": schema.String{Enum: []string{
				ConnectionTypeGeocodeAddress,
				ConnectionTypeGeocodeAutocomplete,
				ConnectionTypeGeocodeNetwork,
				ConnectionTypeGeocodeTiles,
				ConnectionTypeGeocodeTimezone,
				ConnectionTypeImage,
				ConnectionTypeUserPayment,
			}},
			"providerId": schema.String{Enum: []string{
				ConnectionProviderGeocodeAddress,
				ConnectionProviderGeocodeAutocomplete,
				ConnectionProviderGeocodeNetwork,
				ConnectionProviderGeocodeTiles,
				ConnectionProviderGeocodeTimezone,
				ConnectionProviderGiphy,
				ConnectionProviderStripe,
				ConnectionProviderStripeConnect,
				ConnectionProviderUnsplash,
			}},
			"vault":  schema.Object{Wildcard: schema.String{}},
			"data":   schema.Object{Wildcard: schema.String{}},
			"active": schema.Boolean{},
		},
	}
}

/******************************************
 * Getter/Setter Interfaces
 ******************************************/

func (connection *Connection) GetPointer(name string) (any, bool) {

	switch name {

	case "providerId":
		return &connection.ProviderID, true

	case "type":
		return &connection.Type, true

	case "data":
		return &connection.Data, true

	case "vault":
		return &connection.Vault, true

	case "active":
		return &connection.Active, true
	}

	return nil, false
}

func (connection Connection) GetStringOK(name string) (string, bool) {

	switch name {

	case "connectionId":
		return connection.ConnectionID.Hex(), true
	}

	return "", false
}

func (connection *Connection) SetString(name string, value string) bool {
	switch name {

	case "connectionId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			connection.ConnectionID = objectID
			return true
		}
	}

	return false
}
