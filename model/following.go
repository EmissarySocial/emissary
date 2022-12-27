package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/digit"
	"github.com/benpate/rosetta/null"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FollowMethodActivityPub represents the ActivityPub subscription
const FollowMethodActivityPub = "ACTIVITYPUB"

// FollowMethodPoll represents a subscription that must be polled for updates
const FollowMethodPoll = "POLL"

// FollowMethodRSSCloud represents an RSS-Cloud subscription
const FollowMethodRSSCloud = "RSSCLOUD"

// FollowMethodWebSub represents a WebSub subscription
const FollowMethodWebSub = "WEBSUB"

// FollowingStatusNew represents a new following that has not yet been polled
const FollowingStatusNew = "NEW"

// FollowingStatusLoading represents a following that is currently loading
const FollowingStatusLoading = "LOADING"

// FollowingStatusPending represents a following that has been partially connected (e.g. WebSub)
const FollowingStatusPending = "PENDING"

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
	URL           string             `path:"url"            json:"url"            bson:"url"`           // Human-Facing URL that is being followed.
	Links         []digit.Link       `path:"links"          json:"links"          bson:"links"`         // List of links can be used to update this following.
	Method        string             `path:"method"         json:"method"         bson:"method"`        // Method used to update this feed (POLL, WEBSUB, RSS-CLOUD, ACTIVITYPUB)
	Secret        string             `path:"secret"         json:"secret"         bson:"secret"`        // Secret used to authenticate this feed (if required)
	Status        string             `path:"status"         json:"status"         bson:"status"`        // Status of the last poll of Following (NEW, WAITING, SUCCESS, FAILURE)
	StatusMessage string             `path:"statusMessage"  json:"statusMessage"  bson:"statusMessage"` // Optional message describing the status of the last poll
	LastPolled    int64              `path:"lastPolled"     json:"lastPolled"     bson:"lastPolled"`    // Unix Timestamp of the last date that this resource was retrieved.
	PollDuration  int                `path:"pollDuration"   json:"pollDuration"   bson:"pollDuration"`  // Time (in hours) to wait between polling this resource.
	NextPoll      int64              `path:"nextPoll"       json:"nextPoll"       bson:"nextPoll"`      // Unix Timestamp of the next time that this resource should be polled.
	PurgeDuration int                `path:"purgeDuration"  json:"purgeDuration"  bson:"purgeDuration"` // Time (in days) to wait before purging old messages
	ErrorCount    int                `path:"errorCount"     json:"errorCount"     bson:"errorCount"`    // Number of times that this "following" has failed to load (for exponential backoff)

	journal.Journal `path:"journal" json:"-" bson:"journal"`
}

// NewFollowing returns a fully initialized Following object
func NewFollowing() Following {
	return Following{
		FollowingID:   primitive.NewObjectID(),
		Status:        FollowingStatusNew,
		Method:        FollowMethodPoll,
		PollDuration:  24, // default poll interval is 24 hours
		PurgeDuration: 14, // default purge interval is 14 days
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
			"method":        schema.String{Required: true, Enum: []string{FollowMethodPoll, FollowMethodWebSub, FollowMethodRSSCloud, FollowMethodActivityPub}},
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

/*******************************************
 * Other Methods
 *******************************************/

func (following *Following) Origin() OriginLink {
	return OriginLink{
		InternalID: following.FollowingID,
		Label:      following.Label,
		Type:       following.Method,
		URL:        following.URL,
	}
}

func (following *Following) GetLink(property string, value string) digit.Link {
	for _, link := range following.Links {
		if link.GetString(property) == value {
			return link
		}
	}
	return digit.Link{}
}
