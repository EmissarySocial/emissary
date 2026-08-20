package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ConnectionSchema returns the rosetta schema that describes a Connection
func ConnectionSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"connectionId": schema.String{Format: "objectId"},
			"type": schema.String{Enum: []string{
				ConnectionTypeNetwork,
				ConnectionTypeGeocodeAddress,
				ConnectionTypeGeocodeAutocomplete,
				ConnectionTypeGeocodeNetwork,
				ConnectionTypeGeocodeTiles,
				ConnectionTypeGeocodeTimezone,
				ConnectionTypeImage,
				ConnectionTypeUserPayment,
			}},
			"providerId": schema.String{Enum: []string{
				ConnectionProviderBluesky,
				ConnectionProviderGeocodeAddress,
				ConnectionProviderGeocodeAutocomplete,
				ConnectionProviderGeocodeNetwork,
				ConnectionProviderGeocodeTiles,
				ConnectionProviderGeocodeTimezone,
				ConnectionProviderGiphy,
				// ConnectionProviderStripe,
				ConnectionProviderStripeConnect,
				ConnectionProviderUnsplash,
			}},
			// vault holds secrets (API keys, tokens); it uses unsafe-any so the no-html default
			// does not strip characters or collapse whitespace and silently corrupt a secret.
			"vault":  schema.Object{Wildcard: schema.String{Format: "unsafe-any", MaxLength: 8192}},
			"data":   schema.Object{Wildcard: schema.String{MaxLength: 4096}},
			"active": schema.Boolean{},
		},
	}
}

/******************************************
 * Getter/Setter Interfaces
 ******************************************/

// GetPointer returns a pointer to the named property. Implements schema.PointerGetter.
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

// GetStringOK returns the named property. Implements schema.StringGetter.
func (connection Connection) GetStringOK(name string) (string, bool) {

	switch name {

	case "connectionId":
		return connection.ConnectionID.Hex(), true
	}

	return "", false
}

// SetString writes the named property. Implements schema.StringSetter.
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
