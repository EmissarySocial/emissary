package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const ReactionTypeLike = "LIKE"

const ReactionTypeDislike = "DISLIKE"

const ReactionTypeVote = "VOTE"

type Reaction struct {
	ReactionID primitive.ObjectID `path:"reactionId" json:"reactionId" bson:"_id"`    // Unique identifier for this Reaction
	Method     string             `path:"method"     json:"method"     bson:"method"` // Method of following (e.g. "RSS", "RSSCloud", "ActivityPub".)
	Type       string             `path:"type"       json:"type"       bson:"type"`   // Type of reaction (e.g. "like", "dislike", "favorite", "bookmark", "share", "reply", "repost", "follow", "subscribe", "tag", "flag", "comment", "mention", "react", "rsvpYes", "rsvpNo", "rsvpMaybe", "review")
	Value      string             `path:"value"      json:"value"      bson:"value"`  // Custom value assigned to the reaction (emoji, vote, etc)
	Actor      PersonLink         `path:"actor"      json:"actor"      bson:"actor"`  // Person who is reacting to the Content
	Object     OriginLink         `path:"object"     json:"object"     bson:"object"` // Content that is being reacted to

	journal.Journal `path:"journal" json:"journal" bson:"journal"`
}

func NewReaction() Reaction {
	return Reaction{
		ReactionID: primitive.NewObjectID(),
	}
}

func ReactionSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"reactionId": schema.String{Format: "objectId"},
			"userId":     schema.String{Format: "objectId"},
			"actor":      PersonLinkSchema(),
			"object":     OriginLinkSchema(),
			"method":     schema.String{Enum: []string{FollowMethodPoll, FollowMethodWebSub, FollowMethodActivityPub}},
		},
	}
}

/*******************************************
 * data.Object Interface
 *******************************************/

func (reaction *Reaction) ID() string {
	return reaction.ReactionID.Hex()
}

func (reaction *Reaction) GetObjectID(name string) (primitive.ObjectID, error) {
	switch name {
	case "reactionId":
		return reaction.ReactionID, nil
	}

	return primitive.NilObjectID, derp.NewInternalError("model.reaction.GetObjectID", "Invalid property", name)
}

func (reaction *Reaction) GetString(name string) (string, error) {
	switch name {
	case "method":
		return reaction.Method, nil
	case "type":
		return reaction.Type, nil
	case "value":
		return reaction.Value, nil
	}

	return "", derp.NewInternalError("model.reaction.GetString", "Invalid property", name)
}

func (reaction *Reaction) GetInt(name string) (int, error) {
	return 0, derp.NewInternalError("model.reaction.GetInt", "Invalid property", name)
}

func (reaction *Reaction) GetInt64(name string) (int64, error) {
	return 0, derp.NewInternalError("model.reaction.GetInt64", "Invalid property", name)
}

func (reaction *Reaction) GetBool(name string) (bool, error) {
	return false, derp.NewInternalError("model.reaction.GetBool", "Invalid property", name)
}
