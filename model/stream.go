package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Stream corresponds to a top-level path on any Domain.
type Stream struct {
	StreamID primitive.ObjectID `json:"sectionId" bson:"_id"`   // Internal identifier.  Not used publicly
	Token    string             `json:"token"     bson:"token"` // Unique value that identifies this element in the URL
	Label    string             `json:"label"     bson:"label"` // Label used in auto-generated navigation
	Content  []Content          `json:"content" bson:"content"` // Array of content objects in this stream.

	journal.Journal `json:"journal" bson:"journal"`
}

// ID returns the primary key of this object
func (stream *Stream) ID() string {
	return stream.StreamID.Hex()
}
