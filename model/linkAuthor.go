package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type AuthorLink struct {
	InternalID   primitive.ObjectID `path:"internalId"   json:"internalId"   bson:"internalId,omitempty"`   // Unique ID of a document in this database
	Name         string             `path:"name"         json:"name"         bson:"name,omitempty"`         // Name of the author
	ProfileURL   string             `path:"profileUrl"   json:"profileUrl"   bson:"profileUrl,omitempty"`   // URL of the author's profile
	EmailAddress string             `path:"emailAddress" json:"emailAddress" bson:"emailAddress,omitempty"` // Email address of the author
	ImageURL     string             `path:"photoUrl"     json:"photoUrl"     bson:"photoUrl,omitempty"`     // URL of the author's profile photo
	UpdateDate   int64              `path:"updateDate"   json:"updateDate"   bson:"updateDate,omitempty"`   // Unix timestamp of the date/time when this author was last updated.
}

func NewAuthorLink() AuthorLink {
	return AuthorLink{}
}

// Link returns a Link to this author
func (author AuthorLink) Link() Link {

	return Link{
		Relation:   LinkRelationAuthor,
		InternalID: author.InternalID,
		URL:        author.ProfileURL,
		Label:      author.Name,
		UpdateDate: author.UpdateDate,
	}
}
