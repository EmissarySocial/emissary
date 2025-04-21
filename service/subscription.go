package service

import (
	"iter"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/schema"
	"github.com/davecgh/go-spew/spew"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Subscription defines a service that manages all content subscriptions created and imported by Users.
type Subscription struct {
	merchantAccountService *MerchantAccount
	collection             data.Collection
}

// NewSubscription returns a fully initialized Subscription service
func NewSubscription() Subscription {
	return Subscription{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Subscription) Refresh(collection data.Collection, merchantAccountService *MerchantAccount) {
	service.collection = collection
	service.merchantAccountService = merchantAccountService
}

// Close stops any background processes controlled by this service
func (service *Subscription) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *Subscription) Count(criteria exp.Expression) (int64, error) {
	return service.collection.Count(notDeleted(criteria))
}

// Query returns an slice of allthe Subscriptions that match the provided criteria
func (service *Subscription) Query(criteria exp.Expression, options ...option.Option) ([]model.Subscription, error) {
	result := make([]model.Subscription, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)

	return result, err
}

// List returns an iterator containing all of the Subscriptions that match the provided criteria
func (service *Subscription) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the Subscription records that match the provided criteria
func (service *Subscription) Range(criteria exp.Expression, options ...option.Option) (iter.Seq[model.Subscription], error) {

	iter, err := service.List(criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Subscription.Range", "Error creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewSubscription), nil
}

// Load retrieves an Subscription from the database
func (service *Subscription) Load(criteria exp.Expression, subscription *model.Subscription) error {

	if err := service.collection.Load(notDeleted(criteria), subscription); err != nil {
		return derp.Wrap(err, "service.Subscription.Load", "Error loading Subscription", criteria)
	}

	return nil
}

// Save adds/updates an Subscription in the database
func (service *Subscription) Save(subscription *model.Subscription, note string) error {

	// Validate the value before saving
	if err := service.Schema().Validate(subscription); err != nil {
		return derp.Wrap(err, "service.Subscription.Save", "Error validating Subscription", subscription)
	}

	// Validate the Merchant Account
	merchantAccount := model.NewMerchantAccount()
	if err := service.merchantAccountService.LoadByID(subscription.UserID, subscription.MerchantAccountID, &merchantAccount); err != nil {
		return derp.Wrap(err, "service.Subscription.Save", "Error loading Merchant Account", subscription.MerchantAccountID)
	}

	if err := service.merchantAccountService.RefreshAPIKeys(&merchantAccount); err != nil {
		return derp.Wrap(err, "service.Subscription.Save", "Error validating Merchant Account", subscription.MerchantAccountID)
	}

	subscription.MerchantAccountType = merchantAccount.Type

	// Save the subscription to the database
	if err := service.collection.Save(subscription, note); err != nil {
		return derp.Wrap(err, "service.Subscription.Save", "Error saving Subscription", subscription, note)
	}

	return nil
}

// Delete removes an Subscription from the database (virtual delete)
func (service *Subscription) Delete(subscription *model.Subscription, note string) error {

	// Delete this Subscription
	if err := service.collection.Delete(subscription, note); err != nil {
		return derp.Wrap(err, "service.Subscription.Delete", "Error deleting Subscription", subscription, note)
	}

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Subscription) ObjectType() string {
	return "Subscription"
}

// New returns a fully initialized model.Subscription as a data.Object.
func (service *Subscription) ObjectNew() data.Object {
	result := model.NewSubscription()
	return &result
}

func (service *Subscription) ObjectID(object data.Object) primitive.ObjectID {

	if mention, ok := object.(*model.Subscription); ok {
		return mention.SubscriptionID
	}

	return primitive.NilObjectID
}

func (service *Subscription) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *Subscription) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewSubscription()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Subscription) ObjectSave(object data.Object, comment string) error {
	if subscription, ok := object.(*model.Subscription); ok {
		return service.Save(subscription, comment)
	}
	return derp.NewInternalError("service.Subscription.ObjectSave", "Invalid Object Type", object)
}

func (service *Subscription) ObjectDelete(object data.Object, comment string) error {
	if subscription, ok := object.(*model.Subscription); ok {
		return service.Delete(subscription, comment)
	}
	return derp.NewInternalError("service.Subscription.ObjectDelete", "Invalid Object Type", object)
}

func (service *Subscription) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.NewUnauthorizedError("service.Subscription", "Not Authorized")
}

func (service *Subscription) Schema() schema.Schema {
	return schema.New(model.SubscriptionSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *Subscription) QueryAsLookupCodes(userID primitive.ObjectID) ([]form.LookupCode, error) {

	// Query Subscription for this User
	criteria := exp.Equal("userId", userID)
	subscriptions, err := service.Query(criteria)

	if err != nil {
		return nil, derp.Wrap(err, "service.Subscription.QueryAsLookupCodes", "Error querying Subscriptions", criteria)
	}

	spew.Dump(criteria, subscriptions)

	// Map the Subscriptions into LookupCodes
	result := make([]form.LookupCode, 0)

	for _, subscription := range subscriptions {
		result = append(result, subscription.LookupCode())
	}

	return result, nil

}

func (service *Subscription) LoadByID(userID primitive.ObjectID, subscriptionID primitive.ObjectID, subscription *model.Subscription) error {

	criteria := exp.Equal("_id", subscriptionID).
		AndEqual("userId", userID)

	return service.Load(criteria, subscription)
}

func (service *Subscription) LoadByToken(userID primitive.ObjectID, token string, subscription *model.Subscription) error {

	if subscriptionID, err := primitive.ObjectIDFromHex(token); err == nil {
		return service.LoadByID(userID, subscriptionID, subscription)
	} else {
		return derp.Wrap(err, "service.Subscriber.LoadByToken", "Invalid Token", token)
	}
}
