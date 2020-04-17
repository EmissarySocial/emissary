package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Page is a Domain Object where Actors collaborate
type Page struct {
	PageID     primitive.ObjectID `json:"pageId"     bson:"_id"`        // Unique ID of this page
	Properties string             `json:"properties" bson:"properties"` // Page-level data is stored here.  This is opaque to the server, and may be JSON-encoded or encrypted by the client.

	journal.Journal `json:"journal" bson:"journal"`
}

// ID returns the primary key for this record
func (page *Page) ID() string {
	return page.PageID.Hex()
}
