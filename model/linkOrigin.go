package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OriginTypeActivityPub identifies a link was created by an ActivityPub source
const OriginTypeActivityPub = "ACTIVITYPUB"

// OriginTypeInternal identifies a link was created by this application
const OriginTypeInternal = "INTERNAL"

// OriginTypePoll identifies a link was created by an RSS source
const OriginTypePoll = "POLL"

// OriginTypeRSSCloud identifies a link was created by an RSS Cloud source
const OriginTypeRSSCloud = "RSS-CLOUD"

// OriginTypeTwitter identifies a link was created by Twitter
const OriginTypeTwitter = "TWITTER"

const OriginTypeWebSub = "WEBSUB"

// OriginLink represents the original source of a stream that has been imported into Emissary.
// This could be an external ActivityPub server, RSS Feed, or Tweet.
type OriginLink struct {
	InternalID primitive.ObjectID `path:"internalId"  json:"internalId"  bson:"internalId,omitempty"` // Unique ID of a document in this database
	Type       string             `path:"type"        json:"type"        bson:"type"`                 // The type of service that generated this document (RSS, RSS-CLOUD, ACTIVITYPUB, TWITTER, etc.)
	URL        string             `path:"url"         json:"url"         bson:"url"`                  // Public URL of the origin
	Label      string             `path:"label"       json:"label"       bson:"label,omitempty"`      // Human-Friendly label of the origin
	ImageURL   string             `path:"imageUrl"    json:"imageUrl"    bson:"imageUrl,omitempty"`   // URL of the cover image for this document's image
}

// NewOriginLink returns a fully initialized OriginLink
func NewOriginLink() OriginLink {
	return OriginLink{}
}

// OriginLinkSchema returns a JSON Schema for OriginLink structures
func OriginLinkSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"internalId": schema.String{Format: "objectId"},
			"type":       schema.String{Enum: []string{OriginTypeActivityPub, OriginTypeInternal, OriginTypePoll, OriginTypeRSSCloud, OriginTypeTwitter}},
			"url":        schema.String{Format: "url"},
			"label":      schema.String{},
			"summary":    schema.String{},
			"imageUrl":   schema.String{Format: "url"},
			"updateDate": schema.Integer{},
		},
	}
}

// IsEmpty returns TRUE if this OriginLink is empty
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

// Icon returns the standard icon label for this origin
func (origin OriginLink) Icon() string {

	switch origin.Type {

	case OriginTypeActivityPub:
		return "code-slash"
	case OriginTypeInternal:
		return "star"
	case OriginTypePoll:
		return "rss"
	case OriginTypeTwitter:
		return "twitter"
	case OriginTypeWebSub:
		return "websub"
	}

	return "question-square " + origin.Type
}
