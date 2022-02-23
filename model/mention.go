package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Mention struct {
	MentionID primitive.ObjectID `json:"mentionId" bson:"_id"`
	StreamID  primitive.ObjectID `json:"streamId" bson:"streamId"`
	Source    string             `json:"source" bson:"source"`

	journal.Journal `json:"journal" bson:"journal"`
}

func NewMention() Mention {
	return Mention{
		MentionID: primitive.NewObjectID(),
	}
}

func (mention Mention) ID() string {
	return mention.MentionID.Hex()
}
