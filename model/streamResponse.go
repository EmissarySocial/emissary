package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/schema"
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

func StreamResponseSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"streamResponseId": schema.String{Format: "objectId", Required: true},
			"streamId":         schema.String{Format: "objectId", Required: true},
			"actor":            PersonLinkSchema(),
			"origin":           OriginLinkSchema(),
			"type":             schema.String{Enum: []string{ResponseTypeLike, ResponseTypeDislike, ResponseTypeMention, ResponseTypeRepost}, Required: true},
			"value":            schema.String{MaxLength: 64},
		},
	}
}
