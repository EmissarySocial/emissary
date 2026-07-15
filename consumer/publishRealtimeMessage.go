package consumer

import (
	"github.com/EmissarySocial/emissary/realtime"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PublishRealtimeMessage delivers a realtime (SSE) nudge to every browser watching
// the identified object.  Publishers send it post-commit with queue.WithInline():
// the realtime broker holds live sockets, so the task MUST run in this process —
// a stored or cross-process run would nudge nobody.
func PublishRealtimeMessage(factory *service.Factory, args mapof.Any) queue.Result {

	const location = "consumer.PublishRealtimeMessage"

	log.Trace().Msg("Task: PublishRealtimeMessage")

	// Parse the target object from the task arguments
	objectID, err := primitive.ObjectIDFromHex(args.GetString("objectId"))

	if err != nil {
		return queue.Failure(derp.Wrap(err, location, "Invalid 'objectId' argument", args))
	}

	// Build the realtime message for this topic
	message, err := messageForTopic(args.GetInt("topic"), objectID, args.GetString("data"))

	if err != nil {
		return queue.Failure(derp.Wrap(err, location, "Invalid 'topic' argument", args))
	}

	// Deliver directly to this process's broker (synchronous, mutex-safe)
	factory.RealtimeBroker().Send(message)
	return queue.Success()
}

// messageForTopic builds the realtime.Message for a topic, honoring each topic's
// event-name and payload conventions (see realtime/message.go).  The data argument
// is only used by the inbox-activity topics; other topics carry a fixed payload.
func messageForTopic(topic int, objectID primitive.ObjectID, data string) (realtime.Message, error) {

	switch topic {

	case realtime.TopicUpdated:
		return realtime.NewMessage_Updated(objectID), nil

	case realtime.TopicChildUpdated:
		return realtime.NewMessage_ChildUpdated(objectID), nil

	case realtime.TopicNewReplies:
		return realtime.NewMessage_NewReplies(objectID), nil

	case realtime.TopicImportProgress:
		return realtime.NewMessage_ImportProgress(objectID), nil

	case realtime.TopicFollowingUpdated:
		return realtime.NewMessage_FollowingUpdated(objectID), nil

	case realtime.TopicInboxActivity:
		return realtime.NewMessage_InboxActivity(objectID, data), nil

	case realtime.TopicInboxActivity_DirectMessage:
		return realtime.NewMessage_InboxActivity_DirectMessage(objectID, data), nil

	case realtime.TopicInboxActivity_DirectMessage_MLS:
		return realtime.NewMessage_InboxActivity_DirectMessage_MLS(objectID, data), nil

	case realtime.TopicNotification:
		return realtime.NewMessage_Notification(objectID), nil
	}

	return realtime.Message{}, derp.Internal("consumer.messageForTopic", "Unrecognized realtime topic", topic)
}
