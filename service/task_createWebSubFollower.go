package service

import (
	"strconv"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/domain"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/mapof"
	"github.com/labstack/gommon/random"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TaskCreateWebSubFollower struct {
	followerService *Follower
	locatorService  Locator
	ObjectType      string
	ObjectID        primitive.ObjectID
	Format          string // JSONFeed, RSS, Atom
	Mode            string
	Topic           string
	Callback        string
	Secret          string
	LeaseSeconds    int
}

func NewTaskCreateWebSubFollower(followerService *Follower, locatorService Locator, objectType string, objectID primitive.ObjectID, format string, mode string, topic string, callback string, secret string, leaseSeconds int) TaskCreateWebSubFollower {
	return TaskCreateWebSubFollower{
		followerService: followerService,
		locatorService:  locatorService,
		ObjectType:      objectType,
		ObjectID:        objectID,
		Format:          format,
		Mode:            mode,
		Topic:           topic,
		Callback:        callback,
		Secret:          secret,
		LeaseSeconds:    leaseSeconds,
	}
}

func (task TaskCreateWebSubFollower) MarshalMap() map[string]any {
	return mapof.Any{
		"type":         "createWebSubFollower",
		"host":         task.followerService.host,
		"objectType":   task.ObjectType,
		"objectID":     task.ObjectID,
		"format":       task.Format,
		"mode":         task.Mode,
		"topic":        task.Topic,
		"callback":     task.Callback,
		"secret":       task.Secret,
		"leaseSeconds": task.LeaseSeconds,
	}
}

func (task TaskCreateWebSubFollower) Priority() int {
	return 20
}

func (task TaskCreateWebSubFollower) RetryMax() int {
	return 12 // 4096 minutes = 68 hours ~= 3 days
}

func (task TaskCreateWebSubFollower) Run() error {

	switch task.Mode {

	case "subscribe":
		return task.subscribe()

	case "unsubscribe":
		return task.unsubscribe()
	}

	return derp.NewInternalError("service.TaskCreateWebSubFollower.Run", "Invalid mode", task.Mode)
}

func (task TaskCreateWebSubFollower) Hostname() string {
	return domain.NameOnly(task.followerService.host)
}

// subscribe creates/updates a follower record
func (task TaskCreateWebSubFollower) subscribe() error {

	const location = "service.TaskCreateWebSubFollower.subscribe"

	// Calculate lease time (within bounds)
	minLeaseSeconds := 60 * 60 * 24 * 1  // Minimum lease is 1 day
	maxLeaseSeconds := 60 * 60 * 24 * 30 // Maximum lease is 30 days

	if task.LeaseSeconds < minLeaseSeconds {
		task.LeaseSeconds = minLeaseSeconds
	}

	if task.LeaseSeconds > maxLeaseSeconds {
		task.LeaseSeconds = maxLeaseSeconds
	}

	// Create a new Follower record
	follower, err := task.followerService.LoadOrCreateByWebSub(task.ObjectType, task.ObjectID, task.Callback)

	if err != nil {
		return derp.Wrap(err, location, "Error loading follower", task.ObjectID, task.Callback)
	}

	// Set additional properties that are not handled by LoadOrCreateByWebSub
	follower.StateID = model.FollowerStateActive
	follower.Format = task.Format
	follower.ExpireDate = time.Now().Add(time.Second * time.Duration(task.LeaseSeconds)).Unix()
	follower.Data = mapof.Any{
		"secret": task.Secret,
	}

	// Validate the request with the client
	if err := task.validate(&follower); err != nil {
		return derp.Wrap(err, location, "Error validating request", follower.ID)
	}

	// Save the new/updated follower
	if err := task.followerService.Save(&follower, "Created via WebSub"); err != nil {
		return derp.Wrap(err, location, "Error saving follower", follower.ID)
	}

	// Oh yeah...
	return nil
}

// unsubscribe removes a follower record
func (task TaskCreateWebSubFollower) unsubscribe() error {

	const location = "service.TaskCreateWebSubFollower.unsubscribe"

	// Load the existing follower record
	follower := model.NewFollower()
	if err := task.followerService.LoadByWebSub(task.ObjectType, task.ObjectID, task.Callback, &follower); err != nil {
		return derp.Wrap(err, location, "Error loading follower", task.ObjectID, task.Callback)
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
func (task TaskCreateWebSubFollower) validate(follower *model.Follower) error {

	const location = "service.TaskCreateWebSubFollower.validate"

	var body string

	// Validate the request with the client
	challenge := random.String(42)
	transaction := remote.Get(follower.Actor.InboxURL).
		Query("hub.mode", task.Mode).
		Query("hub.topic", task.Topic).
		Query("hub.challenge", challenge).
		Query("hub.lease_seconds", strconv.Itoa(task.LeaseSeconds)).
		Result(&body)

	if err := transaction.Send(); err != nil {
		return derp.Wrap(err, location, "Error sending verification request", follower.ID)
	}

	if body != challenge {
		return derp.NewBadRequestError(location, "Invalid challenge response", follower.ID)
	}

	// Validate the object in our own database
	objectType, objectID, err := task.locatorService.GetObjectFromURL(task.Topic)

	if err != nil {
		return derp.Wrap(err, location, "Error parsing topic URL", follower.ID)
	}

	if objectType != task.ObjectType {
		return derp.NewBadRequestError(location, "Invalid object type", follower.ID)
	}

	if objectID != task.ObjectID {
		return derp.NewBadRequestError(location, "Invalid object ID", follower.ID)
	}

	return nil
}
