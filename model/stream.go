package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Stream corresponds to a top-level path on any Domain.
type Stream struct {
	StreamID       primitive.ObjectID     `json:"streamId"        bson:"_id"`            // Unique identifier of this Stream.  (NOT USED PUBLICLY)
	ParentID       primitive.ObjectID     `json:"parentId"        bson:"parentId"`       // Unique identifier of the "parent" stream. (NOT USED PUBLICLY)
	Template       string                 `json:"template"        bson:"template"`       // Unique identifier (name) of the Template to use when rendering this Stream in HTML.
	Token          string                 `json:"token"           bson:"token"`          // Unique value that identifies this element in the URL
	URL            string                 `json:"url"             bson:"url"`            // Unique URL of this Stream.  This duplicates the "token" field a bit, but it (hopefully?) makes access easier.
	Label          string                 `json:"label"           bson:"label"`          // Text to display in lists of streams, probably displayed at top of stream page, too.
	Description    string                 `json:"description"     bson:"description"`    // Brief summary of this stream, used in lists of streams
	ThumbnailImage string                 `json:"thumbnailImage"  bson:"thumbnailImage"` // Image to display next to the stream in lists.
	AuthorID       primitive.ObjectID     `json:"authorId"        bson:"authorId"`       // Unique identifier of the person who created this stream (NOT USED PUBLICLY)
	AuthorName     string                 `json:"authorName"      bson:"authorName"`     // Full name of the person who created this stream
	AuthorImage    string                 `json:"authorImage"     bson:"authorImage"`    // URL of an image to use for the person who created this stream
	AuthorURL      string                 `json:"authorURL"       bson:"authorURL"`      // URL address of the person who created this stream
	Tags           []string               `json:"tags"            bson:"tags"`           // Organizational Tags
	Data           map[string]interface{} `json:"data"            bson:"data"`           // Set of data to populate into the Template.  This is validated by the JSON-Schema of the Template.
	Source         StreamSourceType       `json:"source"          bson:"source"`         // Identifies the remote source
	SourceID       primitive.ObjectID     `json:"sourceId"        bson:"sourceId"`       // Internal identifier of the source configuration that generated this stream
	SourceURL      string                 `json:"sourceURL"       bson:"sourceURL"`      // URL of the original document published by the source server
	PublishDate    int64                  `json:"publishDate"     bson:"publishDate"`    // Unix timestamp of the date/time when this document is/was/will be first available on the domain.
	UnPublishDate  int64                  `json:"unpublishDate"   bson:"unpublishDate"`  // Unix timestemp of the date/time when this document will no longer be available on the domain.

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

	if stream.Label != other.Label {
		stream.Label = other.Label
		changed = true
	}

	if stream.ThumbnailImage != other.ThumbnailImage {
		stream.ThumbnailImage = other.ThumbnailImage
		changed = true
	}

	if stream.Description != other.Description {
		stream.Description = other.Description
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
