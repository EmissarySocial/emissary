package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PushSubscription represents a single browser's Web Push subscription for a local User.  One User
// may hold several subscriptions (laptop + phone).  Populated from the JSON that the browser's
// PushManager.subscribe() returns.
type PushSubscription struct {
	PushSubscriptionID primitive.ObjectID `bson:"_id"`                 // Unique ID for this subscription
	UserID             primitive.ObjectID `bson:"userId"`              // Owner (local User)
	Endpoint           string             `bson:"endpoint"`            // Push-service URL (unique per browser registration)
	P256DH             string             `bson:"p256dh"`              // Client public key (base64url)
	Auth               string             `bson:"auth"`                // Client auth secret (base64url)
	UserAgent          string             `bson:"userAgent,omitempty"` // User-Agent header (for a future settings list)

	journal.Journal `json:"-" bson:",inline"`
}

// NewPushSubscription returns a fully initialized PushSubscription object
func NewPushSubscription() PushSubscription {
	return PushSubscription{
		PushSubscriptionID: primitive.NewObjectID(),
	}
}

/******************************************
 * data.Object Interface
 ******************************************/

// ID returns a string representation of the PushSubscription's unique id.
func (sub PushSubscription) ID() string {
	return sub.PushSubscriptionID.Hex()
}
