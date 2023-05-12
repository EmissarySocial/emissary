package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Response struct {
	ResponseID primitive.ObjectID `json:"responseId" bson:"_id"`      // Unique identifier for this Response
	Type       string             `json:"type"       bson:"type"`     // Type of Response (e.g. "like", "dislike", "favorite", "bookmark", "share", "reply", "repost", "follow", "subscribe", "tag", "flag", "comment", "mention", "react", "rsvpYes", "rsvpNo", "rsvpMaybe", "review")
	Value      string             `json:"value"      bson:"value"`    // Custom value assigned to the response (emoji, vote, etc)
	Actor      PersonLink         `json:"actor"      bson:"actor"`    // Actor who responded to the Document
	Origin     OriginLink         `json:"origin"     bson:"origin"`   // Original document where the Actor posted their response
	ObjectID   primitive.ObjectID `json:"objectId"   bson:"objectId"` // Internal ID of the Document that the Actor resonded to (Zero for external documents)
	Object     DocumentLink       `json:"object"     bson:"object"`   // Document that the Actor responded to

	journal.Journal `json:"journal" bson:"journal"`
}

func NewResponse() Response {
	return Response{
		ResponseID: primitive.NewObjectID(),
	}
}

/******************************************
 * data.Object Interface
 ******************************************/

func (response *Response) ID() string {
	return response.ResponseID.Hex()
}

func (response *Response) GetJSONLD() mapof.Any {
	return mapof.Any{
		"@context": vocab.ContextTypeActivityStreams,
		"@type":    response.ActivityPubType(),
		"type":     response.ActivityPubType(),
		"actor":    response.Actor.GetJSONLD(),
		"object":   response.Object.URL,
	}
}

/******************************************
 * Other Data Methods
 ******************************************/

func (response Response) ActivityPubType() string {

	switch response.Type {

	case ResponseTypeLike:
		return vocab.ActivityTypeLike

	case ResponseTypeDislike:
		return vocab.ActivityTypeDislike

	default:
		return vocab.ActivityTypeAnnounce
	}
}

// FromLocalActor returns TRUE if this Response was created by a local Actor
func (response Response) FromLocalActor() bool {
	return response.Actor.UserID.IsZero()
}

// FromRemoteActor returns TRUE if this Response was created by a remote Actor
func (response Response) FromRemoteActor() bool {
	return !response.Actor.UserID.IsZero()
}

// ToLocalDocument returns TRUE if this Response was created for a local Document
func (response Response) ToLocalDocument() bool {
	return !response.ObjectID.IsZero()
}

// ToRemoteDocument returns TRUE if this Response was created for a remote Document
func (response Response) ToRemoteDocument() bool {
	return response.ObjectID.IsZero()
}
