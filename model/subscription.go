package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Subscription struct {
	SubscriptionID  primitive.ObjectID `json:"subscriptionId" bson:"_id"`            // Unique Identifier of this record
	ParentStreamID  primitive.ObjectID `json:"parentStreamId" bson:"parentStreamId"` // ID of the stream that owns this subscription
	Method          string             `json:"method"         bson:"method"`         // Method used to subscribe to remote streams (RSS, etc)
	URL             string             `json:"url"            bson:"url"`            // Connection URL for obtaining new sub-streams.
	LastPolled      int64              `json:"lastPolled"     bson:"lastPolled"`     // Unix Timestamp of the last date that this resource was retrieved.
	PollDuration    int                `json:"pollDuration"   bson:"pollDuration"`   // Time (in minutes) to wait between polling this resource.
	journal.Journal `json:"-" bson:"journal"`
}

func NewSubscription() *Subscription {
	return &Subscription{}
}

// ID returns a string represenation of the unique ID for this record.
func (sub *Subscription) ID() string {
	return sub.SubscriptionID.Hex()
}
