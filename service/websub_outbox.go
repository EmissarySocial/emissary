package service

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/iterators"
	"github.com/benpate/derp"
	websubmodel "github.com/benpate/websub/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// WebSubOutbox is an facade for the Follower service that presents a websub Store
type WebSubOutbox struct {
	followerService *Follower
	locatorService  Locator
	parentID        primitive.ObjectID
}

func NewWebSubOutbox(followerService *Follower, locatorService Locator, parentID primitive.ObjectID) WebSubOutbox {
	return WebSubOutbox{
		followerService: followerService,
		locatorService:  locatorService,
		parentID:        parentID,
	}
}

// All returns all subscriptions for the specified topic.
func (outbox WebSubOutbox) All(topic string) ([]websubmodel.Subscription, error) {

	const location = "service.WebSubOutbox.All"

	it, err := outbox.followerService.ListWebSub(outbox.parentID)

	if err != nil {
		return nil, derp.Wrap(err, location, "Failed to load followers", topic)
	}

	result := iterators.Map(it, model.NewFollower, outbox.toSubscription)

	if len(result) == 0 {
		return result, derp.NewNotFoundError(location, "No subscriptions found for topic", topic)
	}

	return result, nil
}

// For returns the subscriptions for the specified callback
func (outbox WebSubOutbox) For(callback string) ([]websubmodel.Subscription, error) {

	const location = "service.WebSubOutbox.All"

	it, err := outbox.followerService.ListWebSubByCallback(outbox.parentID, callback)

	if err != nil {
		return nil, derp.Wrap(err, location, "Failed to load followers", callback)
	}

	result := iterators.Map(it, model.NewFollower, outbox.toSubscription)

	if len(result) == 0 {
		return result, derp.NewNotFoundError(location, "No subscriptions found for callback", callback)
	}

	return result, nil
}

// Add saves/adds a subscription to the store.
func (outbox WebSubOutbox) Add(sub websubmodel.Subscription) error {

	const location = "service.WebSubOutbox.Add"

	_, objectID, err := outbox.locatorService.GetObjectFromURL(sub.Topic)

	if err != nil {
		return derp.Wrap(err, location, "Failed to get object from URL", sub)
	}

	if objectID != outbox.parentID {
		return derp.NewBadRequestError(location, "Topic does not match parent object", sub.Topic, outbox.parentID)
	}

	follower := outbox.fromSubscription(sub)

	// Save the new follower
	if err := outbox.followerService.Save(&follower, "WebSub Add"); err != nil {
		return derp.Wrap(err, location, "Failed to save follower", sub)
	}

	return nil
}

// Get retrieves a subscription given a topic and callback.
func (outbox WebSubOutbox) Get(topic, callback string) (*websubmodel.Subscription, error) {

	const location = "service.WebSubOutbox.Get"

	follower := model.NewFollower()

	if err := outbox.followerService.LoadByWebSub(outbox.parentID, callback, &follower); err != nil {
		return nil, derp.Wrap(err, location, "Failed to load follower", topic, callback)
	}

	result := outbox.toSubscription(follower)
	return &result, nil
}

// Remove removes a subscription from the store.
func (outbox WebSubOutbox) Remove(sub websubmodel.Subscription) error {

	follower := model.NewFollower()

	if err := outbox.followerService.LoadByWebSub(outbox.parentID, sub.Callback, &follower); err != nil {
		return derp.Wrap(err, "service.WebSubOutbox.Remove", "Failed to load follower", sub.Topic, sub.Callback)
	}

	if err := outbox.followerService.Delete(&follower, "Deleted by WebSub"); err != nil {
		return derp.Wrap(err, "service.WebSubOutbox.Remove", "Failed to delete follower", sub.Topic, sub.Callback)
	}

	return nil
}

func (outbox WebSubOutbox) toSubscription(follower model.Follower) websubmodel.Subscription {

	return websubmodel.Subscription{
		ID:       follower.Data.GetInt64("id"),
		Topic:    follower.Actor.ProfileURL,
		Callback: follower.Data.GetString("callback"),
		Secret:   follower.Data.GetString("secret"),
		Expires:  time.Unix(follower.ExpireDate, 0),
	}
}

func (outbox WebSubOutbox) fromSubscription(sub websubmodel.Subscription) model.Follower {

	follower := model.NewFollower()

	follower.ParentID = outbox.parentID
	follower.Method = model.FollowMethodWebSub
	follower.Actor.ProfileURL = sub.Callback
	follower.Data.SetString("callback", sub.Callback)
	follower.Data.SetString("secret", sub.Secret)
	follower.ExpireDate = sub.Expires.Unix()

	return follower
}
