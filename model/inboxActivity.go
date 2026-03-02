package model

import (
	"encoding/json"

	"github.com/benpate/data/journal"
	"github.com/benpate/derp"
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
	IsPublic        bool               `bson:"isPublic"`      // Whether this activity was addressed to the public (i.e. "as:Public")
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

// GetJSONLD returns the original JSON-LD document that was received by the server for this InboxActivity
func (inboxActivity InboxActivity) GetJSONLD() mapof.Any {
	return inboxActivity.RawActivity
}

// NotPublic returns true if this activity is not addressed to the public (i.e. "as:Public")
func (inboxActivity InboxActivity) NotPublic() bool {
	return !inboxActivity.IsPublic
}

// String returns the RawActivity of this InboxActivity as a JSON string
func (inboxActivity InboxActivity) String() string {

	// Marshal the activity as JSON
	data, err := json.Marshal(inboxActivity.RawActivity)

	// Report errors (this should never happen)
	if err != nil {
		derp.Report(derp.Wrap(err, "model.InboxActivity.String", "Unable to marshal RawActivity (this should never happen)", inboxActivity.RawActivity))
	}

	// Success. Always success.
	return string(data)
}
