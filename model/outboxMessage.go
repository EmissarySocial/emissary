package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OutboxMessage represents a single item in a User's inbox or outbox.  It is loosely modelled on the OutboxMessageStreams
// standard, and can be converted into a strict go-fed streams.Type object.
type OutboxMessage struct {
	OutboxMessageID primitive.ObjectID `bson:"_id"`          // Unique ID of the OutboxMessage
	ActorID         primitive.ObjectID `bson:"actorId"`      // Unique ID of the User who owns this OutboxMessage (in their inbox or outbox)
	ActorType       string             `bson:"actorType"`    // Type of the parent object (User or Stream)
	ActorURL        string             `bson:"actorUrl"`     // URL of the parent object (User or Stream)
	ActivityType    string             `bson:"activityType"` // Type of the activity (Create, Follow, Like, Block, etc.)
	ActivityURL     string             `bson:"activityUrl"`  // URL of the ActivityPub object (if applicable)
	ObjectID        string             `bson:"objectId"`     // URL of the object (if applicable)
	Permissions     Permissions        `bson:"permissions"`  // List of permissions for this OutboxMessage

	journal.Journal `bson:",inline"`
}

// NewOutboxMessage returns a fully initialized OutboxMessage record
func NewOutboxMessage() OutboxMessage {
	return OutboxMessage{
		OutboxMessageID: primitive.NewObjectID(),
		Permissions:     NewPermissions(),
	}
}

func OutboxMessageFields() []string {
	return []string{"objectId", "createDate"}
}

func (summary OutboxMessage) Fields() []string {
	return OutboxMessageFields()
}

/******************************************
 * JSONLDGetter Interface
 ******************************************/

func (message OutboxMessage) ActivityPubURL() string {

	if message.ActivityURL != "" {
		return message.ActivityURL
	}

	return message.ActorURL + "/pub/outbox/" + message.OutboxMessageID.Hex()
}

func (message OutboxMessage) GetJSONLD() mapof.Any {

	result := mapof.Any{
		vocab.AtContext:         vocab.ContextTypeActivityStreams,
		vocab.PropertyID:        message.ActivityPubURL(),
		vocab.PropertyActor:     message.ActorURL,
		vocab.PropertyType:      message.ActivityType,
		vocab.PropertyObject:    message.ObjectID,
		vocab.PropertyPublished: message.Created(),
	}

	if message.Permissions.IsAnonymous() {
		result[vocab.PropertyTo] = []string{vocab.NamespaceActivityStreamsPublic}
	} else {
		result[vocab.PropertyTo] = []string{}
	}

	return result
}

func (message OutboxMessage) Created() int64 {
	return message.CreateDate
}

/******************************************
 * data.Object Interface
 ******************************************/

func (message OutboxMessage) ID() string {
	return message.OutboxMessageID.Hex()
}
