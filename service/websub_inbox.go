package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/iterators"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
	websubmodel "meow.tf/websub/model"
)

// WebSubInbox is a facade for the Following service that presents a websub Store
type WebSubInbox struct {
	userID           primitive.ObjectID
	followingService *Following
}

func NewWebSubInbox(followingService *Following, userID primitive.ObjectID) WebSubInbox {
	return WebSubInbox{
		userID:           userID,
		followingService: followingService,
	}
}

// All returns all subscriptions for the specified topic.
func (inbox WebSubInbox) All(topic string) ([]websubmodel.Subscription, error) {

	const location = "service.WebSubInbox.All"

	it, err := inbox.followingService.ListWebSubByTopic(inbox.userID, topic)

	if err != nil {
		return nil, derp.Wrap(err, location, "Failed to load followers", topic)
	}

	result := iterators.Map(it, model.NewFollowing, inbox.toSubscription)

	if len(result) == 0 {
		return result, derp.NewNotFoundError(location, "No subscriptions found for topic", topic)
	}

	return result, nil
}

// For returns the subscriptions for the specified callback
func (inbox WebSubInbox) For(callback string) ([]websubmodel.Subscription, error) {

	const location = "service.WebSubInbox.All"

	it, err := inbox.followingService.ListWebSubByCallback(inbox.userID, callback)

	if err != nil {
		return nil, derp.Wrap(err, location, "Failed to load followers", callback)
	}

	result := iterators.Map(it, model.NewFollowing, inbox.toSubscription)

	if len(result) == 0 {
		return result, derp.NewNotFoundError(location, "No subscriptions found for callback", callback)
	}

	return result, nil
}

// Add saves/adds a subscription to the store.
func (inbox WebSubInbox) Add(sub websubmodel.Subscription) error {
	return derp.NewInternalError("service.WebSubInbox.Add", "Not implemented")
}

// Get retrieves a subscription given a topic and callback.
func (inbox WebSubInbox) Get(topic, callback string) (*websubmodel.Subscription, error) {

	const location = "service.WebSubInbox.Get"

	following := model.NewFollowing()

	if err := inbox.followingService.LoadByWebSub(inbox.userID, topic, callback, &following); err != nil {
		return nil, derp.Wrap(err, location, "Failed to load following", topic, callback)
	}

	result := inbox.toSubscription(following)
	return &result, nil
}

// Remove removes a subscription from the store.
func (inbox WebSubInbox) Remove(sub websubmodel.Subscription) error {

	following := model.NewFollowing()

	if err := inbox.followingService.LoadByWebSub(inbox.userID, sub.Topic, sub.Callback, &following); err != nil {
		return derp.Wrap(err, "service.WebSubInbox.Remove", "Failed to load following", sub.Topic, sub.Callback)
	}

	if err := inbox.followingService.Delete(&following, "Deleted by WebSub"); err != nil {
		return derp.Wrap(err, "service.WebSubInbox.Remove", "Failed to delete following", sub.Topic, sub.Callback)
	}

	return nil
}

func (inbox WebSubInbox) toSubscription(following model.Following) websubmodel.Subscription {
	return websubmodel.Subscription{
		ID:       following.Data.GetInt64("id"),
		Topic:    following.URL,
		Callback: following.Data.GetString("callback"),
		Secret:   following.Data.GetString("secret"),
	}
}
