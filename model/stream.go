package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Stream corresponds to a top-level path on any Domain.
type Stream struct {
	StreamID    primitive.ObjectID     `json:"sectionId" bson:"_id"`           // Internal identifier.  Not used publicly
	Token       string                 `json:"token"     bson:"token"`         // Unique value that identifies this element in the URL
	Title       string                 `json:"title"     bson:"title"`         // Text to display in lists of streams, probably displayed at top of stream page, too.
	Icon        string                 `json:"icon"      bson:"icon"`          // Image to display next to the stream in lists.
	Summary     string                 `json:"summary"   bson:"summary"`       // Brief summary of this stream, used in lists of streams
	Author      string                 `json:"author"    bson:"author"`        // Full name of the person who created this stream
	Tags        []string               `json:"tags"      bson:"tags"`          // Organizational Tags
	Source      StreamSourceType       `json:"source"    bson:"source"`        // Identifies the remote
	SourceID    primitive.ObjectID     `json:"sourceId"  bson:"sourceId"`      // Internal identifier of the source configuration that generated this stream
	SourceURL   string                 `json:"sourceURL" bson:"sourceURL"`     // URL of the original document published by the source server
	Data        map[string]interface{} `json:"data"      bson:"data"`          // Array of content objects in this stream.
	PublishDate int64                  `json:"publishDate" bson:"publishDate"` // Unix timestamp of the date/time when this document was first published.

	journal.Journal `json:"journal" bson:"journal"`
}

// ID returns the primary key of this object
func (stream *Stream) ID() string {
	return stream.StreamID.Hex()
}

// NewStream returns a fully initialized Stream object.
func NewStream() Stream {

	return Stream{
		StreamID: primitive.NewObjectID(),
		Tags:     []string{},
		Data:     map[string]interface{}{},
	}
}

// UpdateWith compares/updates the stream in the arguments with current values.  If any values have been changed, then this function returns TRUE
func (stream *Stream) UpdateWith(other *Stream) bool {

	changed := false

	if stream.Title != other.Title {
		stream.Title = other.Title
		changed = true
	}

	if stream.Icon != other.Icon {
		stream.Icon = other.Icon
		changed = true
	}

	if stream.Summary != other.Summary {
		stream.Summary = other.Summary
		changed = true
	}

	if stream.Author != other.Author {
		stream.Author = other.Author
		changed = true
	}

	/* TODO Comparisons for arrays of strings, and for map[string]interface{}
	if stream.Tags != other.Tags {
		stream.Tags = other.Tags
		changed = true
	}

	if stream.Data...

	*/

	return changed
}
