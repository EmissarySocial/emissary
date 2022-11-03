package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Mention struct {
	MentionID        primitive.ObjectID `json:"mentionId"        bson:"_id"`
	StreamID         primitive.ObjectID `json:"streamId"         bson:"streamId"`
	OriginURL        string             `json:"sourceUrl"        bson:"sourceUrl"`
	AuthorName       string             `json:"authorName"       bson:"authorName"`
	AuthorEmail      string             `json:"authorEmail"      bson:"authorEmail"`
	AuthorWebsiteURL string             `json:"authorWebsiteUrl" bson:"authorWebsiteUrl"`
	AuthorPhotoURL   string             `json:"authorPhotoUrl"   bson:"authorPhotoUrl"`
	AuthorStatus     string             `json:"authorStatus"     bson:"authorStatus"`
	EntryName        string             `json:"entryName"        bson:"entryName"`
	EntrySummary     string             `json:"entrySummary"     bson:"entrySummary"`
	EntryPhotoURL    string             `json:"entryPhotoUrl"    bson:"entryPhotoUrl"`

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
