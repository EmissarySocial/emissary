package model

import (
	"net/url"

	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Activity represents a single item in a User's inbox or outbox.  It is loosely modelled on the ActivityStreams
// standard, and can be converted into a strict go-fed streams.Type object.
type Activity struct {
	ActivityID  primitive.ObjectID `json:"activityId"   bson:"_id"`                   // Unique ID of the Activity
	UserID      primitive.ObjectID `json:"userId"       bson:"userId"`                // Unique ID of the User who owns this Activity (in their inbox or outbox)
	Place       ActivityPlace      `json:"place"        bson:"place"`                 // Place where this Activity is represented (e.g. "Inbox", "Outbox")
	Origin      OriginLink         `json:"origin"       bson:"origin,omitempty"`      // Link to the origin of this Activity
	Document    DocumentLink       `json:"document"     bson:"document,omitempty"`    // Document that is the subject of this Activity
	ContentHTML string             `json:"contentHtml"  bson:"contentHtml,omitempty"` // HTML Content of the Activity
	ContentJSON string             `json:"contentJson"  bson:"contentJson,omitempty"` // Original JSON message, used for reprocessing later.

	// Inbox-specific fields
	FolderID primitive.ObjectID `json:"folderId"     bson:"folderId,omitempty"` // Unique ID of the Folder where this Activity is stored
	ReadDate int64              `json:"readDate"     bson:"readDate"`           // Unix timestamp of the date/time when this Activity was read by the user

	journal.Journal `json:"-" bson:"journal"`
}

// NewActivity returns a fully initialized Activity record
func NewActivity() Activity {
	return Activity{
		ActivityID: primitive.NewObjectID(),
		UserID:     primitive.NilObjectID,
		FolderID:   primitive.NilObjectID,
	}
}

// NewInboxActivity returns a fully initialized Activity record, with the Place field set to "Inbox"
func NewInboxActivity() Activity {
	return Activity{
		ActivityID: primitive.NewObjectID(),
		UserID:     primitive.NilObjectID,
		FolderID:   primitive.NilObjectID,
		Place:      ActivityPlaceInbox,
	}
}

// NewOutboxActivity returns a fully initialized Activity record, with the Place field set to "Outbox"
func NewOutboxActivity() Activity {
	return Activity{
		ActivityID: primitive.NewObjectID(),
		UserID:     primitive.NilObjectID,
		FolderID:   primitive.NilObjectID,
		Place:      ActivityPlaceOutbox,
	}
}

// ActivitySchema returns a JSON Schema that describes this object
func ActivitySchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"activityId":  schema.String{Format: "objectId"},
			"userId":      schema.String{Format: "objectId"},
			"origin":      OriginLinkSchema(),
			"document":    DocumentLinkSchema(),
			"contentHtml": schema.String{Format: "html"},
			"contentJson": schema.String{Format: "json"},
			"folderId":    schema.String{Format: "objectId"},
			"readDate":    schema.Integer{BitSize: 64},
		},
	}
}

/*******************************************
 * data.Object Interface
 *******************************************/

func (activity *Activity) ID() string {
	return activity.ActivityID.Hex()
}

/*******************************************
 * Other Methods
 *******************************************/

// UpdateWithFollowing updates the contents of this activity with a Following record
func (activity *Activity) UpdateWithFollowing(following *Following) {
	activity.UserID = following.UserID
	activity.FolderID = following.FolderID
	activity.Origin = following.Origin()
}

// UpdateWithActivity updates the contents of this activity with another Activity record
func (activity *Activity) UpdateWithActivity(other *Activity) {
	activity.Origin = other.Origin
	activity.Document = other.Document
	activity.ContentHTML = other.ContentHTML
}

// Status returns a string indicating whether this activity has been read or not
func (activity *Activity) Status() string {
	if activity.ReadDate == 0 {
		return "Unread"
	}
	return "Read"
}

// IsInternal returns true if this activity is "owned" by
// this server, and is not federated via another server.
func (activity *Activity) IsInternal() bool {
	return !activity.Origin.InternalID.IsZero()
}

// PublishDate returns the date that this activity was published.
func (activity *Activity) PublishDate() int64 {
	return activity.Document.PublishDate
}

// URL returns the parsed, canonical URL for this Activity (as stored in the document)
func (activity *Activity) URL() *url.URL {
	result, _ := url.Parse(activity.Document.URL)
	return result
}
