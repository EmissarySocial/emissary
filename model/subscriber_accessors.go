package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func SubscriberSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"subscriberId":      schema.String{Format: "objectId"},
			"subscriptionId":    schema.String{Format: "objectId"},
			"userId":            schema.String{Format: "objectId"},
			"subscriptionToken": schema.String{MaxLength: 64},
			"fediverseId":       schema.String{MaxLength: 64},
			"stateId":           schema.String{MaxLength: 32, Enum: []string{SubscriberStateActive, SubscriberStateExpired, SubscriberStateCanceled}},
			"startDate":         schema.Integer{},
			"endDate":           schema.Integer{},
			"recurringType":     schema.String{MaxLength: 32, Enum: []string{SubscriptionRecurringTypeOnetime, SubscriptionRecurringTypeWeekly, SubscriptionRecurringTypeMonthly, SubscriptionRecurringTypeYearly}},
		},
	}
}

/*********************************
 * Getter/Setter Interfaces
 *********************************/

func (user *Subscriber) GetPointer(name string) (any, bool) {

	switch name {

	case "subscriptionToken":
		return &user.SubscriptionToken, true

	case "fediverseId":
		return &user.FediverseID, true

	case "stateId":
		return &user.StateID, true

	case "startDate":
		return &user.StartDate, true

	case "endDate":
		return &user.EndDate, true

	case "recurringType":
		return &user.RecurringType, true

	default:
		return nil, false
	}
}

func (user *Subscriber) GetStringOK(name string) (string, bool) {
	switch name {

	case "subscriberId":
		return user.SubscriberID.Hex(), true

	case "subscriptionId":
		return user.SubscriptionID.Hex(), true

	case "userId":
		return user.UserID.Hex(), true

	default:
		return "", false
	}
}

func (user *Subscriber) SetString(name string, value string) bool {

	switch name {

	case "subscriberId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			user.SubscriberID = objectID
			return true
		}

	case "subscriptionId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			user.SubscriptionID = objectID
			return true
		}

	case "userId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			user.UserID = objectID
			return true
		}

	}

	return false
}
