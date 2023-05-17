package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type StreamResponse struct {
	StreamResponseID primitive.ObjectID `json:"streamResponseId" bson:"_id"`      // Unique ID for this record
	StreamID         primitive.ObjectID `json:"streamId"         bson:"streamId"` // Stream that has been responded to
	Actor            PersonLink         `json:"actor"            bson:"actor"`    // External person who has sent a response
	Origin           OriginLink         `json:"origin"           bson:"origin"`   // Origin of the response - where it came from and how we learned about it
	Type             string             `json:"type"             bson:"type"`     // The type of the response (mention, like, dislike, share/repost, etc)
	Value            string             `json:"value"            bson:"value"`    // Additional response value (for emoji, votes, etc)

	journal.Journal `json:"journal" bson:"journal"`
}

func NewStreamResponse() StreamResponse {
	return StreamResponse{
		StreamResponseID: primitive.NewObjectID(),
		Actor:            NewPersonLink(),
		Origin:           NewOriginLink(),
	}
}

func (response StreamResponse) ID() string {
	return response.StreamResponseID.Hex()
}

// Equal evaluates two StreamResponse object for equality.  To be equal they
// must have identical StreamIDs, Actor.ProfileURLs, Types, and Values.
func (response StreamResponse) Equal(other StreamResponse) bool {

	if response.StreamID != other.StreamID {
		return false
	}

	if response.Actor.ProfileURL != other.Actor.ProfileURL {
		return false
	}

	if response.Type != other.Type {
		return false
	}

	if response.Value != other.Value {
		return false
	}

	return true
}
