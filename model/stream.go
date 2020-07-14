package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Stream corresponds to a top-level path on any Domain.
type Stream struct {
	StreamID      primitive.ObjectID     `json:"streamId"      bson:"_id"`           // Unique identifier of this Stream.  (NOT USED PUBLICLY)
	TemplateID    primitive.ObjectID     `json:"templateId"    bson:"templateId"`    // Unique identifier of the Template to use when rendering this Stream in HTML. (NOT USED PUBLICLY)
	URL           string                 `json:"url"           bson:"url"`           // Unique URL of this Stream.  This duplicates the "token" field a bit, but it (hopefully?) makes access easier.
	Token         string                 `json:"token"         bson:"token"`         // Unique value that identifies this element in the URL
	Title         string                 `json:"title"         bson:"title"`         // Text to display in lists of streams, probably displayed at top of stream page, too.
	Image         string                 `json:"image"         bson:"image"`         // Image to display next to the stream in lists.
	Summary       string                 `json:"summary"       bson:"summary"`       // Brief summary of this stream, used in lists of streams
	AuthorID      primitive.ObjectID     `json:"authorId"      bson:"authorId"`      // Unique identifier of the person who created this stream (NOT USED PUBLICLY)
	AuthorName    string                 `json:"authorName"    bson:"authorName"`    // Full name of the person who created this stream
	AuthorURL     string                 `json:"authorURL"     bson:"authorURL"`     // URL address of the person who created this stream
	Tags          []string               `json:"tags"          bson:"tags"`          // Organizational Tags
	Source        StreamSourceType       `json:"source"        bson:"source"`        // Identifies the remote source
	SourceID      primitive.ObjectID     `json:"sourceId"      bson:"sourceId"`      // Internal identifier of the source configuration that generated this stream
	SourceURL     string                 `json:"sourceURL"     bson:"sourceURL"`     // URL of the original document published by the source server
	Data          map[string]interface{} `json:"data"          bson:"data"`          // Set of data to populate into the Template.  This is validated by the JSON-Schema of the Template.
	PublishDate   int64                  `json:"publishDate"   bson:"publishDate"`   // Unix timestamp of the date/time when this document is/was/will be first available on the domain.
	UnPublishDate int64                  `json:"unpublishDate" bson:"unpublishDate"` // Unix timestemp of the date/time when this document will no longer be available on the domain.

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

	if stream.Image != other.Image {
		stream.Image = other.Image
		changed = true
	}

	if stream.Summary != other.Summary {
		stream.Summary = other.Summary
		changed = true
	}

	if stream.AuthorID != other.AuthorID {
		stream.AuthorID = other.AuthorID
		changed = true
	}

	if stream.AuthorName != other.AuthorName {
		stream.AuthorName = other.AuthorName
		changed = true
	}

	if stream.AuthorURL != other.AuthorURL {
		stream.AuthorURL = other.AuthorURL
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
