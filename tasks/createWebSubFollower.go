package tasks

import (
	"strconv"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/maps"
	"github.com/labstack/gommon/random"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CreateWebSubFollower struct {
	followerService *service.Follower
	locatorService  service.Locator
	objectType      string
	objectID        primitive.ObjectID
	mode            string
	topic           string
	callback        string
	secret          string
	leaseSeconds    int
}

func NewCreateWebSubFollower(followerService *service.Follower, locatorService service.Locator, objectType string, objectID primitive.ObjectID, mode string, topic string, callback string, secret string, leaseSeconds int) CreateWebSubFollower {
	return CreateWebSubFollower{
		followerService: followerService,
		locatorService:  locatorService,
		objectType:      objectType,
		objectID:        objectID,
		mode:            mode,
		topic:           topic,
		callback:        callback,
		secret:          secret,
		leaseSeconds:    leaseSeconds,
	}
}

func (task CreateWebSubFollower) Run() error {

	switch task.mode {
	case "subscribe":
		return task.subscribe()
	case "unsubscribe":
		return task.unsubscribe()
	}

	return derp.NewInternalError("tasks.CreateWebSubFollower.Run", "Invalid mode", task.mode)
}

func (task CreateWebSubFollower) subscribe() error {

	const location = "tasks.CreateWebSubFollower.subscribe"

	// Calculate lease time (within bounds)
	minLeaseSeconds := 60 * 60 * 24 * 1  // Minimum lease is 1 day
	maxLeaseSeconds := 60 * 60 * 24 * 30 // Maximum lease is 30 days

	if task.leaseSeconds < minLeaseSeconds {
		task.leaseSeconds = minLeaseSeconds
	}

	if task.leaseSeconds > maxLeaseSeconds {
		task.leaseSeconds = maxLeaseSeconds
	}

	// Create a new Follower record
	follower := model.NewFollower()
	follower.ParentID = task.objectID
	follower.Type = task.objectType
	follower.Method = model.FollowMethodWebSub
	follower.ExpireDate = time.Now().Add(time.Second * time.Duration(task.leaseSeconds)).Unix()
	follower.Actor = model.PersonLink{
		InboxURL: task.callback,
	}
	follower.Data = maps.Map{
		"secret": task.secret,
	}

	// Validate the request with the client
	if err := task.validate(&follower); err != nil {
		return derp.Report(derp.Wrap(err, location, "Error validating request", follower.ID))
	}

	if err := task.followerService.Save(&follower, "Created via WebSub"); err != nil {
		return derp.Report(derp.Wrap(err, location, "Error saving follower", follower.ID))
	}

	return nil
}

func (task CreateWebSubFollower) unsubscribe() error {

	const location = "tasks.CreateWebSubFollower.unsubscribe"

	// Load the existing follower record
	follower := model.NewFollower()
	if err := task.followerService.LoadByWebSub(task.objectID, task.callback, &follower); err != nil {
		return derp.Wrap(err, location, "Error loading follower", task.objectID, task.callback)
	}

	// Verify the request with the callback server
	if err := task.validate(&follower); err != nil {
		return derp.Wrap(err, location, "Error validating request", follower.ID)
	}

	// Remove the follower from the database.
	if err := task.followerService.Delete(&follower, "unsubscribe"); err != nil {
		return derp.Wrap(err, location, "Error deleting follower", follower.ID)
	}

	return nil
}

// validate verifies that the request is valid, for an object that we own, and that the callback server approves of the request.
func (task CreateWebSubFollower) validate(follower *model.Follower) error {

	const location = "tasks.CreateWebSubFollower.validate"

	var body string

	// Validate the request with the client
	challenge := random.String(42)
	transaction := remote.Get(follower.Actor.InboxURL).
		Query("hub.mode", task.mode).
		Query("hub.topic", task.topic).
		Query("hub.challenge", challenge).
		Query("hub.lease_seconds", strconv.Itoa(task.leaseSeconds)).
		Response(&body, nil)

	if err := transaction.Send(); err != nil {
		return derp.Wrap(err, location, "Error sending verification request", follower.ID)
	}

	if body != challenge {
		return derp.NewBadRequestError(location, "Invalid challenge response", follower.ID)
	}

	// Validate the object in our own database
	objectType, objectID, err := task.locatorService.GetObjectFromURL(task.topic)

	if err != nil {
		return derp.Wrap(err, location, "Error parsing topic URL", follower.ID)
	}

	if objectType != task.objectType {
		return derp.NewBadRequestError(location, "Invalid object type", follower.ID)
	}

	if objectID != task.objectID {
		return derp.NewBadRequestError(location, "Invalid object ID", follower.ID)
	}

	return nil
}