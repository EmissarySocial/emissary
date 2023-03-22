package service

import (
	"math"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queue"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/pub"
	"github.com/benpate/hannibal/vocab"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Publisher struct {
	streamService   *Stream
	followerService *Follower
	userService     *User
	queue           *queue.Queue
}

func NewPublisher(streamService *Stream, followerService *Follower, userService *User, queue *queue.Queue) Publisher {
	return Publisher{
		streamService:   streamService,
		followerService: followerService,
		userService:     userService,
		queue:           queue,
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

	// RULE: WebMentions are handled in the "Publish" action step
	// because it requires knowledge about which fields in the stream contain URLs.

	// Success!
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

	// Get the iterator of followers to notify
	followers, err := service.followerService.ChannelByParent(stream.ParentID)

	if err != nil {
		return derp.Wrap(err, "service.Publisher.Publish", "Error loading followers", stream)
	}

	// If the channel is nil, then there are no followers to notify
	if followers == nil {
		return nil
	}

	// Load the ActivityPub Actor for this Stream
	actor, err := service.userService.ActivityPubActor(stream.Document.Author.InternalID)

	if err != nil {
		return derp.Wrap(err, "service.Publisher.Publish", "Error loading actor", stream)
	}

	// Create the document to be sent
	activityStream := stream.GetJSONLD()
	activityStream["type"] = objectType

	// spew.Dump("SENDING ACTIVITY", activityStream)

	for follower := range followers {
		service.queue.Run(pub.SendActivityQueueTask(actor, activityType, activityStream, follower.Actor.ProfileURL))
	}

	return nil
}

// notifyFOllowers_WebSub sends a WebSub notification to all followers
func (service Publisher) notifyFollowers_WebSub(stream *model.Stream) error {

	followers, err := service.followerService.ChannelWebSub(stream.ParentID)

	if err != nil {
		return derp.Wrap(err, "domain.RealtimeBroker.notifyWebSub", "Error loading WebSub followers", stream)
	}

	// If the channel is nil, then there are no followers to notify
	if followers == nil {
		return nil
	}

	// Loop through all followers, and send WebSub messages through the queue

	for follower := range followers {
		service.queue.Run(NewTaskSendWebSubMessage(*stream, follower))
	}

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
