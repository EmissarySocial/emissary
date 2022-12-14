package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/maps"
	"github.com/benpate/rosetta/null"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FollowMethodActivityPub represents the ActivityPub subscription
const FollowMethodActivityPub = "ACTIVITYPUB"

// FollowMethodRSS represents an RSS subscription
const FollowMethodRSS = "RSS"

// FollowMethodRSSCloud represents an RSS-Cloud subscription
// const FollowMethodRSSCloud = "RSS-CLOUD"

// FollowMethodWebSub represents a WebSub subscription
const FollowMethodWebSub = "WEBSUB"

// FollowingStatusNew represents a new following that has not yet been polled
const FollowingStatusNew = "NEW"

// FollowingStatusLoading represents a following that is currently loading
const FollowingStatusLoading = "LOADING"

// FollowingStatusSuccess represents a following that has successfully loaded
const FollowingStatusSuccess = "SUCCESS"

// FollowingStatusFailure represents a following that has failed to load
const FollowingStatusFailure = "FAILURE"

// Following is a model object that represents a user's following to an external data feed.
// Currently, the only supported feed types are: RSS, Atom, and JSON Feed.  Others may be added in the future.
type Following struct {
	FollowingID   primitive.ObjectID `path:"followingId"    json:"followingId"    bson:"_id"`           // Unique Identifier of this record
	UserID        primitive.ObjectID `path:"userId"         json:"userId"         bson:"userId"`        // ID of the stream that owns this "following"
	FolderID      primitive.ObjectID `path:"folderId"       json:"folderId"       bson:"folderId"`      // ID of the folder to put new messages into
	Label         string             `path:"label"          json:"label"          bson:"label"`         // Label of this "following" record
	URL           string             `path:"url"            json:"url"            bson:"url"`           // Connection URL for obtaining new sub-streams.
	Method        string             `path:"method"         json:"method"         bson:"method"`        // Method used to subscribe to remote streams (RSS, etc)
	Data          maps.Map           `path:"data"           json:"data"           bson:"data"`          // Additional data used by the subscription method
	Status        string             `path:"status"         json:"status"         bson:"status"`        // Status of the last poll of Following (NEW, WAITING, SUCCESS, FAILURE)
	StatusMessage string             `path:"statusMessage"  json:"statusMessage"  bson:"statusMessage"` // Optional message describing the status of the last poll
	LastPolled    int64              `path:"lastPolled"     json:"lastPolled"     bson:"lastPolled"`    // Unix Timestamp of the last date that this resource was retrieved.
	PollDuration  int                `path:"pollDuration"   json:"pollDuration"   bson:"pollDuration"`  // Time (in hours) to wait between polling this resource.
	NextPoll      int64              `path:"nextPoll"       json:"nextPoll"       bson:"nextPoll"`      // Unix Timestamp of the next time that this resource should be polled.
	PurgeDuration int                `path:"purgeDuration"  json:"purgeDuration"  bson:"purgeDuration"` // Time (in days) to wait before purging old messages
	ErrorCount    int                `path:"errorCount"     json:"errorCount"     bson:"errorCount"`    // Number of times that this "following" has failed to load (for exponential backoff)
	IsPublic      bool               `path:"isPublic"       json:"isPublic"       bson:"isPublic"   `   // If TRUE, this record is visible publicly

	journal.Journal `path:"journal" json:"-" bson:"journal"`
}

// NewFollowing returns a fully initialized Following object
func NewFollowing() Following {
	return Following{
		FollowingID:   primitive.NewObjectID(),
		PollDuration:  24, // default poll interval is 24 hours
		PurgeDuration: 14, // default purge interval is 14 days
		Data:          make(maps.Map),
	}
}

// FollowingSchema returns a validating schema for Following objects
func FollowingSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"followingId":   schema.String{Format: "objectId"},
			"userId":        schema.String{Format: "objectId"},
			"folderId":      schema.String{Format: "objectId"},
			"label":         schema.String{Required: true, MinLength: 1, MaxLength: 100},
			"url":           schema.String{Format: "url", Required: true, MinLength: 1, MaxLength: 1000},
			"method":        schema.String{Required: true, Enum: []string{FollowMethodRSS, FollowMethodActivityPub}},
			"status":        schema.String{Enum: []string{FollowingStatusLoading, FollowingStatusSuccess, FollowingStatusFailure}},
			"statusMessage": schema.String{MaxLength: 1000},
			"lastPolled":    schema.Integer{Minimum: null.NewInt64(0)},
			"pollDuration":  schema.Integer{Minimum: null.NewInt64(1), Maximum: null.NewInt64(24 * 7)},
			"nextPoll":      schema.Integer{Minimum: null.NewInt64(0)},
			"errorCount":    schema.Integer{Minimum: null.NewInt64(0)},
			"IsPublic":      schema.Boolean{},
		},
	}
}

/*******************************************
 * data.Object Interface
 *******************************************/

// ID returns the primary key of this object
func (following *Following) ID() string {
	return following.FollowingID.Hex()
}

func (following *Following) GetObjectID(name string) (primitive.ObjectID, error) {
	return primitive.NilObjectID, derp.NewInternalError("model.Following.GetObjectID", "Invalid property", name)
}

func (following *Following) GetString(name string) (string, error) {
	return "", derp.NewInternalError("model.Following.GetString", "Invalid property", name)
}

func (following *Following) GetInt(name string) (int, error) {
	return 0, derp.NewInternalError("model.Following.GetInt", "Invalid property", name)
}

func (following *Following) GetInt64(name string) (int64, error) {
	return 0, derp.NewInternalError("model.Following.GetInt64", "Invalid property", name)
}

func (following *Following) GetBool(name string) (bool, error) {
	return false, derp.NewInternalError("model.Following.GetBool", "Invalid property", name)
}

/*******************************************
 * data.Object Interface
 *******************************************/

func (following *Following) Origin() OriginLink {
	return OriginLink{
		InternalID: following.FollowingID,
		Label:      following.Label,
		Type:       following.Method,
		URL:        following.URL,
	}
}
