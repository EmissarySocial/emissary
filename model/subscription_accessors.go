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
			"subscriptionToken": schema.String{MaxLength: 1024},
			"userId":            schema.String{Format: "objectId"},
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

func (user *Subscription) GetPointer(name string) (any, bool) {

	switch name {

	case "subscriptionToken":
		return &user.SubscriptionToken, true

	case "name":
		return &user.Name, true

	case "description":
		return &user.Description, true

	case "price":
		return &user.Price, true

	case "recurringType":
		return &user.RecurringType, true

	case "isFeatured":
		return &user.IsFeatured, true

	default:
		return nil, false
	}
}

func (user *Subscription) GetStringOK(name string) (string, bool) {
	switch name {

	case "subscriptionId":
		return user.SubscriptionID.Hex(), true

	case "merchantAccountId":
		return user.MerchantAccountID.Hex(), true

	case "userId":
		return user.UserID.Hex(), true

	default:
		return "", false
	}
}

func (user *Subscription) SetString(name string, value string) bool {

	switch name {

	case "subscriptionId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			user.SubscriptionID = objectID
			return true
		}

	case "merchantAccountId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			user.MerchantAccountID = objectID
			return true
		}

	case "userId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			user.SubscriptionID = objectID
			return true
		}
	}

	return false
}
