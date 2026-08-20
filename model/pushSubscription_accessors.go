package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PushSubscriptionSchema returns the rosetta schema that describes a PushSubscription
func PushSubscriptionSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"pushSubscriptionId": schema.String{Format: "objectId"},
			"userId":             schema.String{Format: "objectId"},
			"endpoint":           schema.String{Format: "url", Required: true, MaxLength: 2048},
			"p256dh":             schema.String{Required: true, MaxLength: 256},
			"auth":               schema.String{Required: true, MaxLength: 256},
			"userAgent":          schema.String{MaxLength: 512},
		},
	}
}

/******************************************
 * Getter/Setter Interfaces
 ******************************************/

// GetPointer returns a pointer to the named property. Implements schema.PointerGetter.
func (sub *PushSubscription) GetPointer(name string) (any, bool) {
	switch name {

	case "endpoint":
		return &sub.Endpoint, true

	case "p256dh":
		return &sub.P256DH, true

	case "auth":
		return &sub.Auth, true

	case "userAgent":
		return &sub.UserAgent, true
	}

	return nil, false
}

// GetStringOK returns the named property. Implements schema.StringGetter.
func (sub *PushSubscription) GetStringOK(name string) (string, bool) {
	switch name {

	case "pushSubscriptionId":
		return sub.PushSubscriptionID.Hex(), true

	case "userId":
		return sub.UserID.Hex(), true
	}

	return "", false
}

// SetString writes the named property. Implements schema.StringSetter.
func (sub *PushSubscription) SetString(name string, value string) bool {
	switch name {

	case "pushSubscriptionId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			sub.PushSubscriptionID = objectID
			return true
		}

	case "userId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			sub.UserID = objectID
			return true
		}
	}

	return false
}
