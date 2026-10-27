package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PushSubscription represents a single browser's Web Push subscription for a local User.  One User
// may hold several subscriptions (laptop + phone).  Populated from the JSON that the browser's
// PushManager.subscribe() returns.
type PushSubscription struct {
	PushSubscriptionID primitive.ObjectID `json:"pushSubscriptionId" bson:"_id"`                 // Unique ID for this subscription
	UserID             primitive.ObjectID `json:"userId"             bson:"userId"`              // Owner (local User)
	Endpoint           string             `json:"endpoint"           bson:"endpoint"`            // Push-service URL (unique per browser registration)
	P256DH             string             `json:"p256dh"             bson:"p256dh"`              // Client public key (base64url)
	Auth               string             `json:"auth"               bson:"auth"`                // Client auth secret (base64url)
	UserAgent          string             `json:"userAgent"          bson:"userAgent,omitempty"` // User-Agent header (for a future settings list)
	Format             string             `json:"format"             bson:"format,omitempty"`    // Push format: "" for a browser, MOBILE for a mobile app
	OAuthUserTokenID   primitive.ObjectID `json:"oauthUserTokenId"   bson:"oauthUserTokenId"`    // OAuth grant that created a mobile subscription (zero for a browser)

	journal.Journal `json:"-" bson:",inline"`
}

// PushSubscriptionFormatWeb marks a subscription made by Emissary's own browser service worker
const PushSubscriptionFormatWeb = ""

// PushSubscriptionFormatMobile marks a subscription made by a mobile app. Mobile push follows
// Mastodon's pattern, since that is what most mobile clients and their push relays implement.
const PushSubscriptionFormatMobile = "MOBILE"

// IsMobile returns TRUE if this subscription belongs to a mobile app
func (sub PushSubscription) IsMobile() bool {
	return sub.Format == PushSubscriptionFormatMobile
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
