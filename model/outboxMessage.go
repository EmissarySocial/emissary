package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/hannibal/datetime"
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

// OutboxMessageFields returns the database columns that must be loaded to populate an OutboxMessage
func OutboxMessageFields() []string {
	return []string{"objectId", "createDate"}
}

// Fields returns the database columns that must be loaded to populate an OutboxMessage
// It is part of the FieldLister interface
func (summary OutboxMessage) Fields() []string {
	return OutboxMessageFields()
}

/******************************************
 * JSONLDGetter Interface
 ******************************************/

// ActivityPubURL returns the URL that identifies this message to ActivityPub
func (message OutboxMessage) ActivityPubURL() string {

	if message.ActivityURL != "" {
		return message.ActivityURL
	}

	return message.ActorURL + "/pub/outbox/" + message.OutboxMessageID.Hex()
}

// GetJSONLD returns this message as an ActivityStreams activity
// It is part of the JSONLDGetter interface
func (message OutboxMessage) GetJSONLD() mapof.Any {

	result := mapof.Any{
		vocab.AtContext:      vocab.ContextTypeActivityStreams,
		vocab.PropertyID:     message.ActivityPubURL(),
		vocab.PropertyActor:  message.ActorURL,
		vocab.PropertyType:   message.ActivityType,
		vocab.PropertyObject: message.ObjectID,
		// CreateDate is journal MILLISECONDS; ActivityStreams `published` must be an RFC3339 string,
		// not a raw epoch integer. (message.Created() stays millis for internal paging cursors.)
		vocab.PropertyPublished: datetime.FromUnixMilli(message.Created()),
	}

	if message.Permissions.IsAnonymous() {
		result[vocab.PropertyTo] = []string{vocab.NamespacePublic}
	} else {
		result[vocab.PropertyTo] = []string{}
	}

	return result
}

// Created returns the creation date of this message, in Unix milliseconds
// It is part of the JSONLDGetter interface
func (message OutboxMessage) Created() int64 {
	return message.CreateDate
}

/******************************************
 * data.Object Interface
 ******************************************/

// ID returns the unique identifier for this OutboxMessage (in string format)
// It is part of the data.Object interface
func (message OutboxMessage) ID() string {
	return message.OutboxMessageID.Hex()
}
