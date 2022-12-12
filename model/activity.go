package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const ActivityFormatToot = "TOOT"

const ActivityFormatArticle = "ARTICLE"

const ActivityFormatMedia = "MEDIA"

// Activity represents a single item in a User's inbox or outbox.  It is loosely modelled on the ActivityStreams
// standard, and can be converted into a strict go-fed streams.Type object.
type Activity struct {
	ActivityID   primitive.ObjectID `path:"activityId"   json:"activityId"   bson:"_id"`                    // Unique ID of the Activity
	OwnerID      primitive.ObjectID `path:"ownerId"      json:"ownerId"      bson:"ownerId"`                // Unique ID of the User who owns this Activity (in their inbox or outbox)
	FolderID     primitive.ObjectID `path:"folderId"     json:"folderId"     bson:"folderId,omitempty"`     // Unique ID of the Folder where this Activity is stored
	Origin       OriginLink         `path:"origin"       json:"origin"       bson:"origin,omitempty"`       // Link to the origin of this Activity
	Document     DocumentLink       `path:"document"     json:"document"     bson:"document,omitempty"`     // Document that is the subject of this Activity
	ContentHTML  string             `path:"contentHtml"  json:"contentHtml"  bson:"contentHtml,omitempty"`  // HTML content of the Activity
	OriginalJSON string             `path:"originalJson" json:"originalJson" bson:"originalJson,omitempty"` // Original JSON string that was received from the ActivityPub server
	PublishDate  int64              `path:"publishDate"  json:"publishDate"  bson:"publishDate"`            // Date when this Activity was published
	ReadDate     int64              `path:"readDate"     json:"readDate"     bson:"readDate"`               // Unix timestamp of the date/time when this Activity was read by the owner

	journal.Journal `json:"-" bson:"journal"`
}

func NewActivity() Activity {
	return Activity{
		ActivityID: primitive.NewObjectID(),
	}
}

func ActivitySchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"activityId":   schema.String{Format: "objectId"},
			"ownerId":      schema.String{Format: "objectId"},
			"folderId":     schema.String{Format: "objectId"},
			"document":     DocumentLinkSchema(),
			"contentHtml":  schema.String{Format: "html"},
			"originalJson": schema.String{Format: "json"},
			"publishDate":  schema.Integer{},
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
 * schema.DataObject Interface
 *******************************************/

func (activity *Activity) GetObjectID(name string) (primitive.ObjectID, error) {
	switch name {
	case "activityId":
		return activity.ActivityID, nil
	case "ownerId":
		return activity.OwnerID, nil
	case "folderId":
		return activity.FolderID, nil
	default:
		return primitive.NilObjectID, derp.NewInternalError("model.Activity.GetObjectID", "Invalid name", name)
	}
}

func (activity *Activity) GetString(name string) (string, error) {
	switch name {
	case "originalJson":
		return activity.OriginalJSON, nil
	default:
		return "", derp.NewInternalError("model.Activity.GetString", "Invalid name", name)
	}
}

func (activity *Activity) GetBool(name string) (bool, error) {
	return false, derp.NewInternalError("model.Activity.GetInt", "Invalid name", name)
}

func (activity *Activity) GetInt(name string) (int, error) {
	return 0, derp.NewInternalError("model.Activity.GetInt", "Invalid name", name)
}

func (activity *Activity) GetInt64(name string) (int64, error) {
	switch name {
	case "publishDate":
		return activity.PublishDate, nil
	case "readDate":
		return activity.ReadDate, nil
	default:
		return 0, derp.NewInternalError("model.Activity.GetInt64", "Invalid name", name)
	}
}

/*******************************************
 * Other Methods
 *******************************************/

// Format returns a suggestion for how to display this activity
func (activity Activity) Format() string {

	// TODO: Smarter rules here?

	if activity.Document.Label != "" {
		return ActivityFormatArticle
	}

	if activity.Document.ImageURL != "" {
		return ActivityFormatMedia
	}

	return ActivityFormatToot
}

func (activity *Activity) Status() string {
	if activity.ReadDate == 0 {
		return "Unread"
	}
	return "Read"
}
