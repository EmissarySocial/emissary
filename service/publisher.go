package service

import (
	"math"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/pub"
	"github.com/benpate/hannibal/streams"
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

func (service Publisher) Publish(stream *model.Stream, userID primitive.ObjectID) error {

	var activity string
	if stream.PublishDate == 0 {
		activity = "CREATE"
	} else {
		activity = "UPDATE"
	}

	// Get the current User record
	user := model.NewUser()
	if err := service.userService.LoadByID(userID, &user); err != nil {
		return derp.Wrap(err, "service.Publisher.Publish", "Error loading user", userID)
	}

	// RULE: Update the stream (if necessary)
	if err := service.setPublishedData(stream, &user); err != nil {
		return derp.Wrap(err, "service.Publisher.Publish", "Error setting published data", stream.ID)
	}

	// RULE: Send notifications (if necessary)
	if err := service.notifyFollowers(stream, activity); err != nil {
		return derp.Wrap(err, "service.Publisher.Publish", "Error sending notifications", stream)
	}

	return nil
}

func (service Publisher) Unpublish(stream *model.Stream, userID primitive.ObjectID) error {

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
	if err := service.notifyFollowers_ActivityPub(stream, "DELETE"); err != nil {
		return derp.Wrap(err, "service.Publisher.Unpublish", "Error sending ActivityPub messages", stream)
	}

	// Hey-oh!
	return nil
}

// setPublishData marks this stream as "published"
func (service Publisher) setPublishedData(stream *model.Stream, user *model.User) error {

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

// notifyFollowers creates an "outbox-item" `Stream` and sends
// notifications to all followers of the stream's author
func (service Publisher) notifyFollowers(stream *model.Stream, action string) error {

	if err := service.notifyFollowers_ActivityPub(stream, action); err != nil {
		return derp.Wrap(err, "service.Publisher.notifyFollowers", "Error sending ActivityPub notifications", stream)
	}

	if err := service.notifyFollowers_WebSub(stream); err != nil {
		return derp.Wrap(err, "service.Publisher.notifyFollowers", "Error sending WebSub notifications", stream)
	}

	return nil
}

func (service Publisher) notifyFollowers_ActivityPub(stream *model.Stream, action string) error {

	// Load the ActivityPub Actor for this Stream
	actor, err := service.userService.ActivityPubActor(stream.Document.Author.InternalID)

	if err != nil {
		return derp.Wrap(err, "service.Publisher.Publish", "Error loading actor", stream)
	}

	// Get the iterator of followers to notify
	followers, err := service.followerService.ListActivityPubFollowers(stream.ParentID)

	if err != nil {
		return derp.Wrap(err, "service.Publisher.Publish", "Error loading followers", stream)
	}

	// Create the document to be sent
	activityStream := streams.NewDocument(stream.AsActivityStream(), nil)

	switch action {

	case "CREATE":

		follower := model.NewFollower()
		for followers.Next(&follower) {
			derp.Report(pub.SendCreate(actor, activityStream, follower.Actor.ProfileURL))
			follower = model.NewFollower()
		}

	case "UPDATE":

		follower := model.NewFollower()
		for followers.Next(&follower) {
			derp.Report(pub.SendUpdate(actor, activityStream, follower.Actor.ProfileURL))
			follower = model.NewFollower()
		}

	case "DELETE":

		follower := model.NewFollower()
		for followers.Next(&follower) {
			derp.Report(pub.SendDelete(actor, activityStream, follower.Actor.ProfileURL))
			follower = model.NewFollower()
		}

	default:
		return derp.NewInternalError("service.Publisher.notifyFollowers_ActivityPub", "Unknown action", action)
	}

	return nil
}

func (service Publisher) notifyFollowers_WebSub(stream *model.Stream) error {
	return nil
}
