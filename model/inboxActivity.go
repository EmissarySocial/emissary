package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// InboxActivity represents a inboxActivity that was received via the MLS protocol.
// These inboxActivitys are opaque to the server and are simply stored and forwarded
// to MLS clients as requested.
type InboxActivity struct {
	InboxActivityID primitive.ObjectID `bson:"_id"`           // Unique identifier for this InboxActivity
	UserID          primitive.ObjectID `bson:"userId"`        // The user that received this InboxActivity
	Type            string             `bson:"type"`          // The type of InboxActivity (Create, Update, Like, Follow, etc.)
	ActivityID      string             `bson:"activityId"`    // The ID/URL of this InboxActivity
	ActorID         string             `bson:"actorId"`       // The ID/URL of the actor that sent this InboxActivity (e.g. "https://example.com/users/alice")
	ObjectID        string             `bson:"objectId"`      // The ID/URL of the object that this InboxActivity is about (e.g. "https://example.com/posts/12345")
	MediaType       string             `bson:"mediaType"`     // The media type of the content (e.g. "message/mls")
	RawActivity     mapof.Any          `bson:"rawActivity"`   // The original, unprocessed activity received by the server
	PublishedDate   int64              `bson:"publishedDate"` // Unix epoch (in milliseconds) when this InboxActivity was published
	ReceivedDate    int64              `bson:"receivedDate"`  // Unix epoch (in milliseconds) when this InboxActivity was received by the server

	journal.Journal `bson:",inline"`
}

// NewInboxActivity returns a fully initialized InboxActivity with a unique ID
func NewInboxActivity() InboxActivity {
	return InboxActivity{
		InboxActivityID: primitive.NewObjectID(),
	}
}

// ID returns the string version of the InboxActivity's unique identifier
func (inboxActivity InboxActivity) ID() string {
	return inboxActivity.InboxActivityID.Hex()
}

func (inboxActivity InboxActivity) GetJSONLD() mapof.Any {
	return inboxActivity.RawActivity
}
