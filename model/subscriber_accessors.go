package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func SubscriberSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"subscriberId":   schema.String{Format: "objectId"},
			"subscriptionId": schema.String{Format: "objectId"},
			"userId":         schema.String{Format: "objectId"},
			"emailAddress":   schema.String{Format: "email", MaxLength: 128},
			"fediverseId":    schema.String{MaxLength: 64},
		},
	}
}

/*********************************
 * Getter/Setter Interfaces
 *********************************/

func (subscriber *Subscriber) GetPointer(name string) (any, bool) {

	switch name {

	case "emailAddress":
		return &subscriber.EmailAddress, true

	case "fediverseHandle":
		return &subscriber.FediverseHandle, true

	default:
		return nil, false
	}
}

func (subscriber *Subscriber) GetStringOK(name string) (string, bool) {
	switch name {

	case "subscriberId":
		return subscriber.SubscriberID.Hex(), true

	case "subscriptionId":
		return subscriber.SubscriptionID.Hex(), true

	case "userId":
		return subscriber.UserID.Hex(), true

	default:
		return "", false
	}
}

func (subscriber *Subscriber) SetString(name string, value string) bool {

	switch name {

	case "subscriberId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			subscriber.SubscriberID = objectID
			return true
		}

	case "subscriptionId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			subscriber.SubscriptionID = objectID
			return true
		}

	case "userId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			subscriber.UserID = objectID
			return true
		}
	}

	return false
}
