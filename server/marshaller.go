package server

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/queue"
	"github.com/benpate/hannibal/outbox"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Marshaller struct {
	serverFactory *Factory
}

func NewMarshaller(serverFactory *Factory) Marshaller {
	return Marshaller{
		serverFactory: serverFactory,
	}
}

func (marshaller Marshaller) Marshal(task queue.Task) (map[string]any, bool) {

	switch typed := task.(type) {

	case service.TaskCreateWebSubFollower:

		return mapof.Any{
			"type":         "createWebSubFollower",
			"hostname":     typed.Hostname(),
			"objectType":   typed.ObjectType,
			"objectID":     typed.ObjectID,
			"format":       typed.Format,
			"mode":         typed.Mode,
			"topic":        typed.Topic,
			"callback":     typed.Callback,
			"secret":       typed.Secret,
			"leaseSeconds": typed.LeaseSeconds,
		}, true

	case service.TaskReceiveWebMention:
		return mapof.Any{
			"type":     "receiveWebMention",
			"hostname": typed.Hostname(),
			"source":   typed.Source,
			"target":   typed.Target,
		}, true

	case service.TaskSendWebMention:
		return mapof.Any{
			"type":     "sendWebMention",
			"hostname": typed.Hostname(),
			"source":   typed.Source,
			"target":   typed.Target,
		}, true

	case service.TaskSendWebSubMessage:

		return mapof.Any{
			"type":     "sendWebSubMessage",
			"hostname": typed.Hostname(),
			"follower": typed.Follower.MarshalMap(),
		}, true

	case outbox.SendTask:

		return mapof.Any{
			"type":      "hannibal.outbox.sendTask",
			"hostname":  typed.Hostname(),
			"actor":     typed.Actor.ActorID(),
			"message":   typed.Message,
			"recipient": typed.Recipient.ID(),
		}, true
	}

	return nil, false
}

func (marshaller Marshaller) Unmarshal(journal *queue.Journal) bool {

	// Find the host from the task arguments
	hostname := journal.Arguments.GetString("hostname")

	// Find the correct domain factory
	factory, err := marshaller.serverFactory.ByDomainName(hostname)

	if err != nil {
		return false
	}

	switch journal.Arguments.GetString("type") {

	case "createWebSubFollower":

		journal.Task = service.NewTaskCreateWebSubFollower(
			factory.Follower(),
			factory.Locator(),
			journal.Arguments.GetString("objectType"),
			objectID(journal.Arguments.GetString("objectID")),
			journal.Arguments.GetString("format"),
			journal.Arguments.GetString("mode"),
			journal.Arguments.GetString("topic"),
			journal.Arguments.GetString("callback"),
			journal.Arguments.GetString("secret"),
			journal.Arguments.GetInt("leaseSeconds"),
		)

		return true

	case "receiveWebMention":

		journal.Task = service.NewTaskReceiveWebMention(
			factory.Stream(),
			factory.Mention(),
			factory.User(),
			journal.Arguments.GetString("source"),
			journal.Arguments.GetString("target"),
		)

		return true

	case "sendWebMention":

		journal.Task = service.NewTaskSendWebMention(
			journal.Arguments.GetString("source"),
			journal.Arguments.GetString("target"),
		)

		return true

	case "sendWebSubMessage":
		follower := model.NewFollower()
		follower.UnmarshalMap(journal.Arguments.GetMap("follower"))

		journal.Task = service.NewTaskSendWebSubMessage(follower)
		return true

	case "hannibal.outbox.sendTask":

		userID := objectID(journal.Arguments.GetString("actor"))
		if actor, err := factory.User().ActivityPubActor(userID, true); err == nil {
			activityStream := factory.ActivityStream()
			if recipient, err := activityStream.Load(journal.Arguments.GetString("recipient")); err == nil {
				message := journal.Arguments.GetMap("message")
				journal.Task = outbox.NewSendTask(actor, message, recipient)
				return true
			}
		}
	}

	return false
}

func objectID(objectID string) primitive.ObjectID {
	result, _ := primitive.ObjectIDFromHex(objectID)
	return result
}
