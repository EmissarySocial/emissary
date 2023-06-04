package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Response struct {
	ResponseID primitive.ObjectID `json:"responseId" bson:"_id"`     // Unique identifier for this Response
	URL        string             `json:"url"        bson:"url"`     // URL of this Response document
	ActorID    string             `json:"actorId"    bson:"actorId"` // URL of the Actor who made the response
	ObjectID   string             `json:"objectId"   bson:"orginId"` // URL of the Object that the actor responded to
	Type       string             `json:"type"       bson:"type"`    // Type of Response (e.g. "like", "dislike", "favorite", "bookmark", "share", "reply", "repost", "follow", "subscribe", "tag", "flag", "comment", "mention", "react", "rsvpYes", "rsvpNo", "rsvpMaybe", "review")
	Summary    string             `json:"summary"    bson:"summary"` // Summary of the response (e.g. "I liked this post because...")
	Content    string             `json:"value"      bson:"value"`   // Custom value assigned to the response (emoji, vote, etc.)

	journal.Journal `json:"-" bson:",inline"`
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
		"id":       response.URL,
		"type":     response.ActivityPubType(),
		"actor":    response.ActorID,
		"object":   response.ObjectID,
		"summary":  response.Summary,
	}

	if response.Content != "" {
		result["content"] = response.Content
	}

	return result
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

// IsEqual returns TRUE if two responses match urls, actors, objects, types, and values
func (response Response) IsEqual(other Response) bool {
	return (response.URL == other.URL) && (response.ActorID == other.ActorID) && (response.ObjectID == other.ObjectID) && (response.Type == other.Type) && (response.Content == other.Content)
}
