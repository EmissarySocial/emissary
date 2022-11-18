package model

import (
	"time"

	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OriginSourceActivityPub identifies a link was created by an ActivityPub source
const OriginSourceActivityPub = "ACTIVITYPUB"

// OriginSourceInternal identifies a link was created by this application
const OriginSourceInternal = "INTERNAL"

// OriginSourceRSS identifies a link was created by an RSS source
const OriginSourceRSS = "RSS"

// OriginSourceRSSCloud identifies a link was created by an RSS Cloud source
const OriginSourceRSSCloud = "RSS-CLOUD"

// OriginSourceTwitter identifies a link was created by Twitter
const OriginSourceTwitter = "TWITTER"

// OriginLink represents the original source of a stream that has been imported into Emissary.
// This could be an external ActivityPub server, RSS Feed, or Tweet.
type OriginLink struct {
	InternalID primitive.ObjectID `path:"internalId" json:"internalId" bson:"internalId,omitempty"` // Unique ID of a document in this database
	Source     string             `path:"source"     json:"source"     bson:"source"`               // The source that generated this document (RSS, RSS-CLOUD, ACTIVITYPUB, TWITTER, etc.)
	Label      string             `path:"label"      json:"label"      bson:"label,omitempty"`      // Label of the original document
	URL        string             `path:"url"        json:"url"        bson:"url"`                  // Public URL of the original record
	UpdateDate int64              `path:"updateDate" json:"updateDate" bson:"updateDate"`           // Unix timestamp of the date/time when this link was last updated.
}

func NewOriginLink() OriginLink {
	return OriginLink{
		UpdateDate: time.Now().Unix(),
	}
}

func OriginLinkSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"internalId": schema.String{Format: "objectId"},
			"source":     schema.String{Enum: []string{OriginSourceActivityPub, OriginSourceInternal, OriginSourceRSS, OriginSourceRSSCloud, OriginSourceTwitter}},
			"label":      schema.String{},
			"url":        schema.String{Format: "url"},
			"updateDate": schema.Integer{},
		},
	}
}

// Link returns a Link to this origin
func (origin OriginLink) Link() Link {
	return Link{
		InternalID: origin.InternalID,
		Relation:   LinkRelationOriginal,
		Source:     origin.Source,
		URL:        origin.URL,
		Label:      origin.Label,
		UpdateDate: origin.UpdateDate,
	}
}

func (origin OriginLink) Icon() string {
	switch origin.Source {
	case "ACTIVITYPUB":
		return "code-slash"
	case "INTERNAL":
		return "star"
	case "RSS":
		return "rss"
	case "TWITTER":
		return "twitter"
	}
	return "question-square"
}
