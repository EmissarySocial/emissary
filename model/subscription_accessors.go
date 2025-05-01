package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func SubscriptionSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"subscriptionId":    schema.String{Format: "objectId"},
			"merchantAccountId": schema.String{Format: "objectId"},
			"remoteId":          schema.String{MaxLength: 1024},
			"name":              schema.String{MaxLength: 64},
			"description":       schema.String{MaxLength: 256},
			"price":             schema.String{MaxLength: 32},
			"recurringType":     schema.String{MaxLength: 32, Enum: []string{SubscriptionRecurringTypeOnetime, SubscriptionRecurringTypeWeekly, SubscriptionRecurringTypeMonthly, SubscriptionRecurringTypeYearly}},
			"isFeatured":        schema.Boolean{},
		},
	}
}

/*********************************
 * Getter/Setter Interfaces
 *********************************/

func (subscription *Subscription) GetPointer(name string) (any, bool) {

	switch name {

	case "remoteId":
		return &subscription.RemoteID, true

	case "name":
		return &subscription.Name, true

	case "description":
		return &subscription.Description, true

	case "price":
		return &subscription.Price, true

	case "recurringType":
		return &subscription.RecurringType, true

	case "isFeatured":
		return &subscription.IsFeatured, true

	default:
		return nil, false
	}
}

func (subscription *Subscription) GetStringOK(name string) (string, bool) {
	switch name {

	case "subscriptionId":
		return subscription.SubscriptionID.Hex(), true

	case "merchantAccountId":
		return subscription.MerchantAccountID.Hex(), true

	default:
		return "", false
	}
}

func (subscription *Subscription) SetString(name string, value string) bool {

	switch name {

	case "subscriptionId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			subscription.SubscriptionID = objectID
			return true
		}

	case "merchantAccountId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			subscription.MerchantAccountID = objectID
			return true
		}
	}

	return false
}
