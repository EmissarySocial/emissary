package model

// OriginLink represents the original source of a stream that has been imported into Emissary.
// This could be an external ActivityPub server, RSS Feed, or Tweet.
type OriginLink struct {
	Source     string `path:"source"     json:"source"     bson:"source"`          // The source that generated this document
	Label      string `path:"label"      json:"label"      bson:"label,omitempty"` // Label of the original document
	URL        string `path:"url"        json:"url"        bson:"url"`             // Public URL of the original record
	UpdateDate int64  `path:"updateDate" json:"updateDate" bson:"updateDate"`      // Unix timestamp of the date/time when this link was last updated.
}

// Link returns a Link to this origin
func (origin OriginLink) Link() Link {
	return Link{
		Relation:   LinkRelationOriginal,
		Source:     origin.Source,
		URL:        origin.URL,
		Label:      origin.Label,
		UpdateDate: origin.UpdateDate,
	}
}
