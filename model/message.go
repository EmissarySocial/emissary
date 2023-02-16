package model

import (
	"net/url"

	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Message represents a single item in a User's inbox or outbox.  It is loosely modelled on the MessageStreams
// standard, and can be converted into a strict go-fed streams.Type object.
type Message struct {
	MessageID   primitive.ObjectID `json:"activityId"   bson:"_id"`                   // Unique ID of the Message
	UserID      primitive.ObjectID `json:"userId"       bson:"userId"`                // Unique ID of the User who owns this Message (in their inbox or outbox)
	Origin      OriginLink         `json:"origin"       bson:"origin,omitempty"`      // Link to the origin of this Message
	Document    DocumentLink       `json:"document"     bson:"document,omitempty"`    // Document that is the subject of this Message
	ContentHTML string             `json:"contentHtml"  bson:"contentHtml,omitempty"` // HTML Content of the Message
	ContentJSON string             `json:"contentJson"  bson:"contentJson,omitempty"` // Original JSON message, used for reprocessing later.
	FolderID    primitive.ObjectID `json:"folderId"     bson:"folderId,omitempty"`    // Unique ID of the Folder where this Message is stored
	ReadDate    int64              `json:"readDate"     bson:"readDate"`              // Unix timestamp of the date/time when this Message was read by the user

	journal.Journal `json:"-" bson:"journal"`
}

// NewMessage returns a fully initialized Message record
func NewMessage() Message {
	return Message{
		MessageID: primitive.NewObjectID(),
		UserID:    primitive.NilObjectID,
		FolderID:  primitive.NilObjectID,
	}
}

/******************************************
 * data.Object Interface
 ******************************************/

func (activity *Message) ID() string {
	return activity.MessageID.Hex()
}

/******************************************
 * Other Methods
 ******************************************/

// UpdateWithFollowing updates the contents of this activity with a Following record
func (activity *Message) UpdateWithFollowing(following *Following) {
	activity.UserID = following.UserID
	activity.FolderID = following.FolderID
	activity.Origin = following.Origin()
}

// UpdateWithMessage updates the contents of this activity with another Message record
func (activity *Message) UpdateWithMessage(other *Message) {
	activity.Origin = other.Origin
	activity.Document = other.Document
	activity.ContentHTML = other.ContentHTML
}

// Status returns a string indicating whether this activity has been read or not
func (activity *Message) Status() string {
	if activity.ReadDate == 0 {
		return "Unread"
	}
	return "Read"
}

// IsInternal returns true if this activity is "owned" by
// this server, and is not federated via another server.
func (activity *Message) IsInternal() bool {
	return !activity.Origin.InternalID.IsZero()
}

// PublishDate returns the date that this activity was published.
func (activity *Message) PublishDate() int64 {
	return activity.Document.PublishDate
}

// URL returns the parsed, canonical URL for this Message (as stored in the document)
func (activity *Message) URL() *url.URL {
	result, _ := url.Parse(activity.Document.URL)
	return result
}
