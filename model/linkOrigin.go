package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OriginTypeActivityPub identifies a link was created by an ActivityPub source
const OriginTypeActivityPub = "ACTIVITYPUB"

// OriginTypeInternal identifies a link was created by this application
const OriginTypeInternal = "INTERNAL"

// OriginTypeRSS identifies a link was created by an RSS source
const OriginTypeRSS = "RSS"

// OriginTypeRSSCloud identifies a link was created by an RSS Cloud source
const OriginTypeRSSCloud = "RSS-CLOUD"

// OriginTypeTwitter identifies a link was created by Twitter
const OriginTypeTwitter = "TWITTER"

// OriginLink represents the original source of a stream that has been imported into Emissary.
// This could be an external ActivityPub server, RSS Feed, or Tweet.
type OriginLink struct {
	InternalID primitive.ObjectID `path:"internalId"  json:"internalId"  bson:"internalId,omitempty"` // Unique ID of a document in this database
	Type       string             `path:"type"        json:"type"        bson:"type"`                 // The type of service that generated this document (RSS, RSS-CLOUD, ACTIVITYPUB, TWITTER, etc.)
	URL        string             `path:"url"         json:"url"         bson:"url"`                  // Public URL of the origin
	Label      string             `path:"label"       json:"label"       bson:"label,omitempty"`      // Human-Friendly label of the origin
	ImageURL   string             `path:"imageUrl"    json:"imageUrl"    bson:"imageUrl,omitempty"`   // URL of the cover image for this document's image
}

func NewOriginLink() OriginLink {
	return OriginLink{}
}

func OriginLinkSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"internalId": schema.String{Format: "objectId"},
			"type":       schema.String{Enum: []string{OriginTypeActivityPub, OriginTypeInternal, OriginTypeRSS, OriginTypeRSSCloud, OriginTypeTwitter}},
			"url":        schema.String{Format: "url"},
			"label":      schema.String{},
			"summary":    schema.String{},
			"imageUrl":   schema.String{Format: "url"},
			"updateDate": schema.Integer{},
		},
	}
}

func (origin OriginLink) IsEmpty() bool {
	return origin.InternalID.IsZero() && (origin.URL == "")
}

// Link returns a Link to this origin
func (origin OriginLink) Link() Link {
	return Link{
		InternalID: origin.InternalID,
		Relation:   LinkRelationOriginal,
		Source:     origin.Type,
		URL:        origin.URL,
		Label:      origin.Label,
	}
}

func (origin OriginLink) Icon() string {
	switch origin.Type {
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
