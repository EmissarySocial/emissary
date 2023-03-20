package model

import "go.mongodb.org/mongo-driver/bson/primitive"

// Link represents a link to another document on the Internet.
type Link struct {
	Relation   string             `json:"rel"        bson:"rel"`                  // The relationship of the linked document, per https://www.iana.org/assignments/link-relations/link-relations.xhtml
	Source     string             `json:"source"     bson:"source"`               // The source of the link.  This could be "ACTIVITYPUB", "RSS", "TWITTER", or "EMAIL"
	InternalID primitive.ObjectID `json:"internalId" bson:"internalId,omitempty"` // Unique ID of a document in this database
	Label      string             `json:"label"      bson:"label,omitempty"`      // Label of the link
	URL        string             `json:"url"        bson:"url,omitempty"`        // Public URL of the document
	UpdateDate int64              `json:"updateDate" bson:"updateDate"`           // Unix timestamp of the date/time when this link was last updated.
}

// ID implements the Set.ID interface
func (link Link) ID() string {
	return link.Relation
}

// IsEmpty returns TRUE if this link is empty
func (link Link) IsEmpty() bool {
	return (link.URL == "" && link.InternalID.IsZero())
}

// IsPresent returns TRUE if this link has a valid value
func (link Link) IsPresent() bool {
	return !link.IsEmpty()
}

func (link Link) HTML() string {
	if link.URL == "" {
		return ""
	}

	return "<link rel=\"" + link.Relation + "\" href=\"" + link.URL + "\">"
}

func (link Link) Header() string {
	return `<` + link.URL + `>; rel="` + link.Relation + `"`
}
