package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OriginLink represents the original source of a stream that has been imported into Emissary.
// This could be an external ActivityPub server, RSS Feed, or Tweet.
type OriginLink struct {
	InternalID primitive.ObjectID `json:"internalId" bson:"internalId,omitempty"` // Unique ID of a document in this database
	Type       string             `json:"type"       bson:"type"`                 // The type of service that generated this document (RSS, RSS-CLOUD, ACTIVITYPUB, TWITTER, etc.)
	URL        string             `json:"url"        bson:"url"`                  // Public URL of the origin
	Label      string             `json:"label"      bson:"label,omitempty"`      // Human-readable label text of the origin
	Summary    string             `json:"summary" bson:"summary,omitempty"`       // Human-readable summary text of the origin
	ImageURL   string             `json:"imageUrl"   bson:"imageUrl,omitempty"`   // URL of the cover image for this document's image
}

// NewOriginLink returns a fully initialized OriginLink
func NewOriginLink() OriginLink {
	return OriginLink{}
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
		return "activitypub"
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
