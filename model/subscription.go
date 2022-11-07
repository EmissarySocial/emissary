package model

import (
	"time"

	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SubscriptionMethodRSS represents an RSS subscription
const SubscriptionMethodRSS = "RSS"

// SubscriptionMethodWebSub represents a WebSub subscription
const SubscriptionMethodWebSub = "WEBSUB"

type Subscription struct {
	SubscriptionID  primitive.ObjectID `path:"subscriptionId" json:"subscriptionId" bson:"_id"`            // Unique Identifier of this record
	ParentStreamID  primitive.ObjectID `path:"parentStreamId" json:"parentStreamId" bson:"parentStreamId"` // ID of the stream that owns this subscription
	Method          string             `path:"method"         json:"method"         bson:"method"`         // Method used to subscribe to remote streams (RSS, etc)
	Tags            []string           `path:"tags"           json:"tags"           bson:"tags"`           // Tags to apply to all items from this subscription
	URL             string             `path:"url"            json:"url"            bson:"url"`            // Connection URL for obtaining new sub-streams.
	LastPolled      int64              `path:"lastPolled"     json:"lastPolled"     bson:"lastPolled"`     // Unix Timestamp of the last date that this resource was retrieved.
	PollDuration    int                `path:"pollDuration"   json:"pollDuration"   bson:"pollDuration"`   // Time (in hours) to wait between polling this resource.
	NextPoll        int64              `path:"nextPoll"       json:"nextPoll"       bson:"nextPoll"`       // Unix Timestamp of the next time that this resource should be polled.
	journal.Journal `json:"-" bson:"journal"`
}

func NewSubscription() Subscription {
	return Subscription{
		PollDuration: 24, // default poll interval is 24 hours
	}
}

/*******************************************
 * DATA.OBJECT INTERFACE
 *******************************************/

// ID returns the primary key of this object
func (sub *Subscription) ID() string {
	return sub.SubscriptionID.Hex()
}

// MarkPolled updates the lastPolled and nextPoll timestamps.
func (sub *Subscription) MarkPolled() {

	// RULE: Default Poll Duration is 24 hours
	if sub.PollDuration == 0 {
		sub.PollDuration = 24
	}

	// RULE: Require that poll duration is at least 1 hour
	if sub.PollDuration < 1 {
		sub.PollDuration = 1
	}

	// Update poll time stamps
	sub.LastPolled = time.Now().Unix()
	sub.NextPoll = sub.LastPolled + int64(sub.PollDuration*60)
}
