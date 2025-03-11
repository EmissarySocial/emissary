package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OriginLink represents the original source of a stream that has been imported into Emissary.
// This could be an external ActivityPub server, RSS Feed, or Tweet.
type OriginLink struct {
	Type        string             `json:"type"        bson:"type,omitempty"`        // The type of message that this document (DIRECT, LIKE, DISLIKE, REPLY, ANNOUNCE)
	FollowingID primitive.ObjectID `json:"followingId" bson:"followingId,omitempty"` // Unique ID of a document in this database
	Label       string             `json:"label"       bson:"label,omitempty"`       // Human-friendly label of the origin
	URL         string             `json:"url"         bson:"url,omitempty"`         // Public URL of the origin
	IconURL     string             `json:"iconUrl"     bson:"iconUrl,omitempty"`     // URL of the a avatar/icon image for this origin
}

// NewOriginLink returns a fully initialized OriginLink
func NewOriginLink() OriginLink {
	return OriginLink{}
}

// Equals returns TRUE if the URL for this OriginLink is the same as the URL for another OriginLink
func (origin OriginLink) Equals(other OriginLink) bool {
	return origin.URL == other.URL
}

// IsEmpty returns TRUE if this OriginLink is empty
func (origin OriginLink) IsEmpty() bool {
	return origin.FollowingID.IsZero() && (origin.URL == "")
}

func (origin OriginLink) NotEmpty() bool {
	return !origin.IsEmpty()
}

// Icon returns the standard icon label for this origin
func (origin OriginLink) Icon() string {

	switch origin.Type {

	case OriginTypePrimary:
		return "user"

	case OriginTypeReply:
		return "reply"

	case OriginTypeLike:
		return "thumbs-up"

	case OriginTypeDislike:
		return "thumbs-down"

	case OriginTypeAnnounce:
		return "rocket"
	}

	return "question-square"
}
