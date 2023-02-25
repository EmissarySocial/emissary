package service

import (
	"math"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/pub"
	"github.com/benpate/hannibal/vocab"
	"github.com/davecgh/go-spew/spew"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Publisher struct {
	streamService   *Stream
	followerService *Follower
	userService     *User
}

func NewPublisher(streamService *Stream, followerService *Follower, userService *User) Publisher {
	return Publisher{
		streamService:   streamService,
		followerService: followerService,
		userService:     userService,
	}
}

func (service Publisher) Publish(stream *model.Stream, userID primitive.ObjectID, objectType string) error {

	// Determine what we're doing with the published stream (Create/Update/Delete)
	activityType := service.guessActivityType(stream)

	// Get the current User record
	user := model.NewUser()
	if err := service.userService.LoadByID(userID, &user); err != nil {
		return derp.Wrap(err, "service.Publisher.Publish", "Error loading user", userID)
	}

	// RULE: Update the stream
	if err := service.setPublished(stream, &user); err != nil {
		return derp.Wrap(err, "service.Publisher.Publish", "Error setting published data", stream.ID)
	}

	// RULE: Send ActivityPub notifications (if necessary)
	if err := service.notifyFollowers_ActivityPub(stream, activityType, objectType); err != nil {
		return derp.Wrap(err, "service.Publisher.notifyFollowers", "Error sending ActivityPub notifications", stream)
	}

	// RULE: Send WebSub notifications (if necessary)
	if err := service.notifyFollowers_WebSub(stream); err != nil {
		return derp.Wrap(err, "service.Publisher.notifyFollowers", "Error sending WebSub notifications", stream)
	}

	return nil
}

func (service Publisher) Unpublish(stream *model.Stream, userID primitive.ObjectID, objectType string) error {

	// RULE: Set the "UnPublish" date
	stream.UnPublishDate = time.Now().Unix()
	if err := service.streamService.Save(stream, "Un-Publish"); err != nil {
		return derp.Wrap(err, "render.StepPublish", "Error saving stream", stream)
	}

	// Get the current User record
	user := model.NewUser()
	if err := service.userService.LoadByID(userID, &user); err != nil {
		return derp.Wrap(err, "service.Publisher.Publish", "Error loading user", userID)
	}

	// RULE: Send ActivityPub Delete messages to federated peers
	if err := service.notifyFollowers_ActivityPub(stream, vocab.ActivityTypeDelete, objectType); err != nil {
		return derp.Wrap(err, "service.Publisher.Unpublish", "Error sending ActivityPub messages", stream)
	}

	// Hey-oh!
	return nil
}

// setPublished marks this stream as "published"
func (service Publisher) setPublished(stream *model.Stream, user *model.User) error {

	// RULE: IF this stream is not yet published, then set the publish date
	if stream.PublishDate > time.Now().Unix() {
		stream.PublishDate = time.Now().Unix()
	}

	// RULE: Move unpublish date all the way to the end of time.
	// TODO: LOW: May want to set automatic unpublish dates later...
	stream.UnPublishDate = math.MaxInt64

	// RULE: Set Author to the currently logged in user.
	stream.Document.Author = user.PersonLink()

	// Re-save the Stream with the updated values.
	if err := service.streamService.Save(stream, "Publish"); err != nil {
		return derp.Wrap(err, "render.StepPublish", "Error saving stream", stream)
	}

	// Done.
	return nil
}

func (service Publisher) notifyFollowers_ActivityPub(stream *model.Stream, activityType string, objectType string) error {

	// Load the ActivityPub Actor for this Stream
	actor, err := service.userService.ActivityPubActor(stream.Document.Author.InternalID)

	if err != nil {
		return derp.Wrap(err, "service.Publisher.Publish", "Error loading actor", stream)
	}

	// Get the iterator of followers to notify
	followers, err := service.followerService.ListActivityPub(stream.ParentID)

	if err != nil {
		return derp.Wrap(err, "service.Publisher.Publish", "Error loading followers", stream)
	}

	// Create the document to be sent
	activityStream := stream.AsActivityStream()
	activityStream["type"] = objectType

	spew.Dump("SENDING ACTIVITY", activityStream)

	follower := model.NewFollower()
	for followers.Next(&follower) {
		if err := pub.SendActivity(actor, activityType, activityStream, follower.Actor.ProfileURL); err != nil {
			return derp.Wrap(err, "service.Publisher.Publish", "Error sending ActivityPub message", stream)
		}
		follower = model.NewFollower()
	}

	return nil
}

// notifyFOllowers_WebSub sends a WebSub notification to all followers
func (service Publisher) notifyFollowers_WebSub(stream *model.Stream) error {

	/*
		followers, err := service.followerService.ListWebSub(stream.ParentID)

		if err != nil {
			return derp.Wrap(err, "service.Publisher.Publish", "Error loading followers", stream)
		}
	*/

	return nil
}

func (service Publisher) guessActivityType(stream *model.Stream) string {

	if stream.Journal.DeleteDate > 0 {
		return vocab.ActivityTypeDelete
	}

	if stream.PublishDate > time.Now().Unix() {
		return vocab.ActivityTypeCreate
	}

	return vocab.ActivityTypeUpdate
}
