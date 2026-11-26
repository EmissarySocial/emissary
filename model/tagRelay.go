package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TagRelayStatePending means the @application actor has sent a Follow and is waiting for an Accept
const TagRelayStatePending = "PENDING"

// TagRelayStateActive means the relay accepted the Follow and is sending tagged posts
const TagRelayStateActive = "ACTIVE"

// TagRelay records that the @application actor follows a #hashtag relay (such as tags.pub) for this server
type TagRelay struct {
	TagRelayID primitive.ObjectID `json:"tagRelayId" bson:"_id"`      // Unique ID of this TagRelay
	Name       string             `json:"name"       bson:"name"`     // Tag name sent to the relay, lowercase, without the leading #
	Value      string             `json:"value"      bson:"value"`    // Normalized tag (ToToken), matching TagFollow.Value
	ActorURL   string             `json:"actorUrl"   bson:"actorUrl"` // URL of the relay's actor for this tag
	FollowID   string             `json:"followId"   bson:"followId"` // ID of the Follow activity sent to the relay
	State      string             `json:"state"      bson:"state"`    // TagRelayStatePending or TagRelayStateActive

	journal.Journal `json:"-" bson:",inline"`
}

// NewTagRelay returns a fully initialized TagRelay
func NewTagRelay() TagRelay {
	return TagRelay{
		TagRelayID: primitive.NewObjectID(),
		State:      TagRelayStatePending,
	}
}

// ID returns the unique identifier for this TagRelay, implementing the data.Object interface
func (tagRelay TagRelay) ID() string {
	return tagRelay.TagRelayID.Hex()
}
