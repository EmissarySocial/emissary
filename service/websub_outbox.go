package service

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/iterators"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
	websubmodel "meow.tf/websub/model"
)

// WebSubOutbox is an facade for the Follower service that presents a websub Store
type WebSubOutbox struct {
	parentID        primitive.ObjectID
	followerService *Follower
}

func NewWebSubOutbox(followerService *Follower, parentID primitive.ObjectID) WebSubOutbox {
	return WebSubOutbox{
		parentID:        parentID,
		followerService: followerService,
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

	follower := model.NewFollower()
	follower.Method = model.FollowMethodWebSub
	follower.Actor.ProfileURL = sub.Callback
	follower.Data.SetInt64("id", sub.ID)
	follower.Data.SetString("callback", sub.Callback)
	follower.ExpireDate = sub.Expires.Unix()

	return derp.NewInternalError("service.WebSubOutbox.Add", "Not implemented")
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