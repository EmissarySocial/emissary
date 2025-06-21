package consumer

import (
	"strconv"
	"time"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/labstack/gommon/random"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CreateWebSubFollower(factory *domain.Factory, args mapof.Any) queue.Result {

	const location = "consumer.CreateWebSubFollower"

	// Collect Arguments
	objectType := args.GetString("objectType")
	objectID := objectID(args.GetString("objectId"))
	format := args.GetString("format")
	mode := args.GetString("mode")
	topic := args.GetString("topic")
	callback := args.GetString("callback")
	secret := args.GetString("secret")
	leaseSeconds := args.GetInt("leaseSeconds")

	switch mode {

	case "subscribe":
		return createWebSubFollower_subscribe(factory, objectType, objectID, format, mode, topic, callback, secret, leaseSeconds)

	case "unsubscribe":
		return createWebSubFollower_unsubscribe(factory, objectType, objectID, mode, topic, callback, leaseSeconds)
	}

	return queue.Failure(derp.InternalError(location, "Invalid mode", mode))
}

// subscribe creates/updates a follower record
func createWebSubFollower_subscribe(factory *domain.Factory, objectType string, objectID primitive.ObjectID, format string, mode string, topic string, callback string, secret string, leaseSeconds int) queue.Result {

	const location = "consumer.createWebSubFollower_subscribe"

	// Calculate lease time (within bounds)
	minLeaseSeconds := 60 * 60 * 24 * 1  // Minimum lease is 1 day
	maxLeaseSeconds := 60 * 60 * 24 * 30 // Maximum lease is 30 days

	if leaseSeconds < minLeaseSeconds {
		leaseSeconds = minLeaseSeconds
	}

	if leaseSeconds > maxLeaseSeconds {
		leaseSeconds = maxLeaseSeconds
	}

	// Create a new Follower record
	followerService := factory.Follower()
	follower, err := followerService.LoadOrCreateByWebSub(objectType, objectID, callback)

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Error loading follower", objectID, callback))
	}

	// Set additional properties that are not handled by LoadOrCreateByWebSub
	follower.StateID = model.FollowerStateActive
	follower.Format = format
	follower.ExpireDate = time.Now().Add(time.Second * time.Duration(leaseSeconds)).Unix()
	follower.Data = mapof.Any{
		"secret": secret,
	}

	// Validate the request with the client
	if result := createWebSubFollower_validate(factory, &follower, objectType, objectID, mode, topic, leaseSeconds); result.NotSuccessful() {
		return result
	}

	// Save the new/updated follower
	if err := followerService.Save(&follower, "Created via WebSub"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Error saving follower", follower.ID))
	}

	// Oh yeah...
	return queue.Success()
}

// unsubscribe removes a follower record
func createWebSubFollower_unsubscribe(factory *domain.Factory, objectType string, objectID primitive.ObjectID, mode string, topic string, callback string, leaseSeconds int) queue.Result {

	const location = "consumer.createWebSubFollower_unsubscribe"

	// Load the existing follower record
	followerService := factory.Follower()
	follower := model.NewFollower()
	if err := followerService.LoadByWebSub(objectType, objectID, callback, &follower); err != nil {
		return queue.Error(derp.Wrap(err, location, "Error loading follower", objectID, callback))
	}

	// Verify the request with the callback server
	if result := createWebSubFollower_validate(factory, &follower, objectType, objectID, mode, topic, leaseSeconds); result.NotSuccessful() {
		return result
	}

	// Remove the follower from the database.
	if err := followerService.Delete(&follower, "unsubscribe"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Error deleting follower", follower.ID))
	}

	return queue.Success()
}

// validate verifies that the request is valid, for an object that we own, and that the callback server approves of the request.
func createWebSubFollower_validate(factory *domain.Factory, follower *model.Follower, objectType string, objectID primitive.ObjectID, mode string, topic string, leaseSeconds int) queue.Result {

	const location = "consumer.createWebSubFollower_validate"

	var body string

	// Validate the request with the client
	challenge := random.String(42)
	transaction := remote.Get(follower.Actor.InboxURL).
		Query("hub.mode", mode).
		Query("hub.topic", topic).
		Query("hub.challenge", challenge).
		Query("hub.lease_seconds", strconv.Itoa(leaseSeconds)).
		Result(&body)

	if err := transaction.Send(); err != nil {
		return queue.Error(derp.Wrap(err, location, "Error sending verification request", follower.ID))
	}

	if body != challenge {
		return queue.Failure(derp.BadRequestError(location, "Invalid challenge response", follower.ID))
	}

	// Validate the object in our own database
	locatorService := factory.Locator()
	foundObjectType, foundObjectID, err := locatorService.GetObjectFromURL(topic)

	if err != nil {
		return queue.Failure(derp.Wrap(err, location, "Error parsing topic URL", follower.ID))
	}

	if objectType != foundObjectType {
		return queue.Failure(derp.BadRequestError(location, "Invalid object type", follower.ID))
	}

	if objectID != foundObjectID {
		return queue.Failure(derp.BadRequestError(location, "Invalid object ID", follower.ID))
	}

	return queue.Success()
}
