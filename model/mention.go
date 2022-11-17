package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Mention struct {
	MentionID        primitive.ObjectID `json:"mentionId"        bson:"_id"`
	StreamID         primitive.ObjectID `json:"streamId"         bson:"streamId"`
	OriginURL        string             `json:"originUrl"        bson:"originUrl"`
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

func MentionSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"mentionId":        schema.String{Format: "objectId"},
			"streamId":         schema.String{Format: "objectId"},
			"sourceUrl":        schema.String{Format: "uri"},
			"authorName":       schema.String{MaxLength: 50},
			"authorEmail":      schema.String{Format: "email"},
			"authorWebsiteUrl": schema.String{Format: "uri"},
			"authorPhotoUrl":   schema.String{Format: "uri"},
			"authorStatus":     schema.String{MaxLength: 500},
			"entryName":        schema.String{MaxLength: 50},
			"entrySummary":     schema.String{MaxLength: 500},
			"entryPhotoUrl":    schema.String{Format: "uri"},
		},
	}
}

/*******************************************
 * data.Object Interface
 *******************************************/

func (mention Mention) ID() string {
	return mention.MentionID.Hex()
}

func (mention *Mention) GetObjectID(name string) (primitive.ObjectID, error) {

	switch name {
	case "mentionId":
		return mention.MentionID, nil
	case "streamId":
		return mention.StreamID, nil
	}
	return primitive.NilObjectID, derp.NewInternalError("model.Mention.GetObjectID", "Invalid property", name)
}

func (mention *Mention) GetString(name string) (string, error) {
	switch name {
	case "originUrl":
		return mention.OriginURL, nil
	case "authorName":
		return mention.AuthorName, nil
	case "authorEmail":
		return mention.AuthorEmail, nil
	case "authorWebsiteUrl":
		return mention.AuthorWebsiteURL, nil
	case "authorPhotoUrl":
		return mention.AuthorPhotoURL, nil
	case "authorStatus":
		return mention.AuthorStatus, nil
	case "entryName":
		return mention.EntryName, nil
	case "entrySummary":
		return mention.EntrySummary, nil
	case "entryPhotoUrl":
		return mention.EntryPhotoURL, nil
	}
	return "", derp.NewInternalError("model.Mention.GetString", "Invalid property", name)
}

func (mention *Mention) GetInt(name string) (int, error) {
	return 0, derp.NewInternalError("model.Mention.GetInt", "Invalid property", name)
}

func (mention *Mention) GetInt64(name string) (int64, error) {
	return 0, derp.NewInternalError("model.Mention.GetInt64", "Invalid property", name)
}
