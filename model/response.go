package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Response struct {
	ResponseID primitive.ObjectID `json:"responseId" bson:"_id"`     // Unique identifier for this Response
	Type       string             `json:"type"       bson:"type"`    // Type of Response (e.g. "like", "dislike", "favorite", "bookmark", "share", "reply", "repost", "follow", "subscribe", "tag", "flag", "comment", "mention", "react", "rsvpYes", "rsvpNo", "rsvpMaybe", "review")
	Actor      PersonLink         `json:"actor"      bson:"actor"`   // Actor who responded
	Message    DocumentLink       `json:"message"    bson:"message"` // Message that the Actor responded to
	Value      string             `json:"value"      bson:"value"`   // Custom value assigned to the response (emoji, vote, etc)

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

func (response Response) ID() string {
	return response.ResponseID.Hex()
}

/******************************************
 * Other Data Methods
 ******************************************/

func (response Response) GetJSONLD() mapof.Any {
	result := mapof.Any{
		"@context": vocab.ContextTypeActivityStreams,
		"id":       response.ActivityPubID(),
		"type":     response.ActivityPubType(),
		"actor":    response.Actor.ProfileURL,
		"object":   response.Message.URL,
	}

	if response.Value != "" {
		result["content"] = response.Value
	}

	return result
}

func (response Response) ActivityPubID() string {

	switch response.Type {

	case ResponseTypeLike:
		return response.Actor.ProfileURL + "/likes/" + response.ResponseID.Hex()

	case ResponseTypeDislike:
		return response.Actor.ProfileURL + "/dislikes/" + response.ResponseID.Hex()

	default:
		return response.Actor.ProfileURL + "/mentions/" + response.ResponseID.Hex()
	}
}

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
