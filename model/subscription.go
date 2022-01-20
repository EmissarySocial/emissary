package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/derp"
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

/*******************************************
 * DATA.OBJECT INTERFACE
 *******************************************/

// ID returns the primary key of this object
func (sub *Subscription) ID() string {
	return sub.SubscriptionID.Hex()
}

// GetPath implements the path.Getter interface, allowing named READ access to specific values
func (sub *Subscription) GetPath(path string) (interface{}, bool) {
	return nil, false
}

// GetPath implements the path.Getter interface, allowing named WRITE access to specific values
func (sub *Subscription) SetPath(path string, value interface{}) error {
	return derp.New(derp.CodeInternalError, "whisper.model.Subscription.GetPath", "unimplemented")
}
