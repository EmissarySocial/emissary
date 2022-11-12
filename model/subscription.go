package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/null"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SubscriptionMethodRSS represents an RSS subscription
const SubscriptionMethodRSS = "RSS"

// SubscriptionMethodWebSub represents a WebSub subscription
const SubscriptionMethodWebSub = "WEBSUB"

// SubscriptionStatusNew represents a new subscription that has not yet been polled
const SubscriptionStatusNew = "NEW"

// SubscriptionStatusLoading represents a subscription that is currently loading
const SubscriptionStatusLoading = "LOADING"

// SubscriptionStatusSuccess represents a subscription that has successfully loaded
const SubscriptionStatusSuccess = "SUCCESS"

// SubscriptionStatusFailure represents a subscription that has failed to load
const SubscriptionStatusFailure = "FAILURE"

// Subscription is a model object that represents a user's subscription to an external data feed.
// Currently, the only supported feed types are: RSS, Atom, and JSON Feed.  Others may be added in the future.
type Subscription struct {
	SubscriptionID primitive.ObjectID `path:"subscriptionId" json:"subscriptionId" bson:"_id"`           // Unique Identifier of this record
	UserID         primitive.ObjectID `path:"userId"         json:"userId"         bson:"userId"`        // ID of the stream that owns this subscription
	InboxFolderID  primitive.ObjectID `path:"inboxFolderId"  json:"inboxFolderId"  bson:"inboxFolderId"` // ID of the inbox folder to put messages into
	Label          string             `path:"label"          json:"label"          bson:"label"`         // Label of this subscription
	URL            string             `path:"url"            json:"url"            bson:"url"`           // Connection URL for obtaining new sub-streams.
	Method         string             `path:"method"         json:"method"         bson:"method"`        // Method used to subscribe to remote streams (RSS, etc)
	Status         string             `path:"status"         json:"status"         bson:"status"`        // Status of the last poll of Subscription (NEW, WAITING, SUCCESS, FAILURE)
	StatusMessage  string             `path:"statusMessage"  json:"statusMessage"  bson:"statusMessage"` // Optional message describing the status of the last poll
	LastPolled     int64              `path:"lastPolled"     json:"lastPolled"     bson:"lastPolled"`    // Unix Timestamp of the last date that this resource was retrieved.
	PollDuration   int                `path:"pollDuration"   json:"pollDuration"   bson:"pollDuration"`  // Time (in hours) to wait between polling this resource.
	NextPoll       int64              `path:"nextPoll"       json:"nextPoll"       bson:"nextPoll"`      // Unix Timestamp of the next time that this resource should be polled.
	ErrorCount     int                `path:"errorCount"     json:"errorCount"     bson:"errorCount"`    // Number of times that this subscription has failed to load (for exponential backoff)

	journal.Journal `path:"journal" json:"-" bson:"journal"`
}

// NewSubscription returns a fully initialized Subscription object
func NewSubscription() Subscription {
	return Subscription{
		SubscriptionID: primitive.NewObjectID(),
		PollDuration:   24, // default poll interval is 24 hours
	}
}

// SubscriptionSchema returns a validating schema for Subscription objects
func SubscriptionSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"subscriptionId": schema.String{Format: "objectId"},
			"userId":         schema.String{Format: "objectId"},
			"inboxFolderId":  schema.String{Format: "objectId"},
			"label":          schema.String{Required: true, MinLength: 1, MaxLength: 100},
			"url":            schema.String{Format: "url", Required: true, MinLength: 1, MaxLength: 1000},
			"method":         schema.String{Required: true, Enum: []string{SubscriptionMethodRSS, SubscriptionMethodWebSub}},
			"status":         schema.String{Enum: []string{SubscriptionStatusLoading, SubscriptionStatusSuccess, SubscriptionStatusFailure}},
			"statusMessage":  schema.String{MaxLength: 1000},
			"lastPolled":     schema.Integer{Minimum: null.NewInt64(0)},
			"pollDuration":   schema.Integer{Minimum: null.NewInt64(1), Maximum: null.NewInt64(24 * 7)},
			"nextPoll":       schema.Integer{Minimum: null.NewInt64(0)},
			"errorCount":     schema.Integer{Minimum: null.NewInt64(0)},
		},
	}
}

/*******************************************
 * data.Object Interface
 *******************************************/

// ID returns the primary key of this object
func (sub *Subscription) ID() string {
	return sub.SubscriptionID.Hex()
}
