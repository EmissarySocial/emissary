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
	ActivityID primitive.ObjectID `path:"activityId"   json:"activityId"   bson:"_id"`                // Unique ID of the Activity
	OwnerID    primitive.ObjectID `path:"ownerId"      json:"ownerId"      bson:"ownerId"`            // Unique ID of the User who owns this Activity (in their inbox or outbox)
	Place      ActivityPlace      `path:"place"        json:"place"        bson:"place"`              // Place where this Activity is represented (e.g. "Inbox", "Outbox")
	Origin     OriginLink         `path:"origin"       json:"origin"       bson:"origin,omitempty"`   // Link to the origin of this Activity
	Document   DocumentLink       `path:"document"     json:"document"     bson:"document,omitempty"` // Document that is the subject of this Activity
	Content    Content            `path:"content"      json:"content"      bson:"content,omitempty"`  // Content of the Activity

	// Inbox-specific fields
	FolderID primitive.ObjectID `path:"folderId"     json:"folderId"     bson:"folderId,omitempty"` // Unique ID of the Folder where this Activity is stored
	ReadDate int64              `path:"readDate"     json:"readDate"     bson:"readDate"`           // Unix timestamp of the date/time when this Activity was read by the owner

	journal.Journal `json:"-" bson:"journal"`
}

// NewActivity returns a fully initialized Activity record
func NewActivity() Activity {
	return Activity{
		ActivityID: primitive.NewObjectID(),
		OwnerID:    primitive.NilObjectID,
		FolderID:   primitive.NilObjectID,
	}
}

// ActivitySchema returns a JSON Schema that describes this object
func ActivitySchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"activityId":   schema.String{Format: "objectId"},
			"ownerId":      schema.String{Format: "objectId"},
			"folderId":     schema.String{Format: "objectId"},
			"document":     DocumentLinkSchema(),
			"contentHtml":  schema.String{Format: "html"},
			"originalJson": schema.String{Format: "json"},
			"readDate":     schema.Integer{},
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
	activity.OwnerID = following.UserID
	activity.FolderID = following.FolderID
	activity.Origin = following.Origin()
}

// UpdateWithActivity updates the contents of this activity with another Activity record
func (activity *Activity) UpdateWithActivity(other *Activity) {
	activity.Origin = other.Origin
	activity.Document = other.Document
	activity.Content = other.Content
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
