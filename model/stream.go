package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/datatype"
	"github.com/benpate/nebula"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Stream corresponds to a top-level path on any Domain.
type Stream struct {
	StreamID        primitive.ObjectID   `path:"streamId"       json:"streamId"        bson:"_id"`                     // Unique identifier of this Stream.  (NOT USED PUBLICLY)
	ParentID        primitive.ObjectID   `path:"parentId"       json:"parentId"        bson:"parentId"`                // Unique identifier of the "parent" stream. (NOT USED PUBLICLY)
	ParentIDs       []primitive.ObjectID `path:"parentIds"      json:"parentIds"       bson:"parentIds"`               // Slice of all parent IDs of this Stream
	TemplateID      string               `path:"templateId"     json:"templateId"      bson:"templateId"`              // Unique identifier (name) of the Template to use when rendering this Stream in HTML.
	StateID         string               `path:"stateId"        json:"stateId"         bson:"stateId"`                 // Unique identifier of the State this Stream is in.  This is used to populate the State information from the Template service at load time.
	Criteria        Criteria             `path:"criteria"       json:"criteria"        bson:"criteria"`                // Criteria for which users can access this stream.
	Token           string               `path:"token"          json:"token"           bson:"token"`                   // Unique value that identifies this element in the URL
	Label           string               `path:"label"          json:"label"           bson:"label"`                   // Text to display in lists of streams, probably displayed at top of stream page, too.
	Description     string               `path:"description"    json:"description"     bson:"description"`             // Brief summary of this stream, used in lists of streams
	AuthorID        primitive.ObjectID   `path:"authorId"       json:"authorId"        bson:"authorId,omitempty"`      // Unique identifier of the person who created this stream (NOT USED PUBLICLY)
	AuthorName      string               `path:"authorName"     json:"authorName"      bson:"authorName,omitempty"`    // Full name of the person who created this stream
	AuthorImage     string               `path:"authorImage"    json:"authorImage"     bson:"authorImage,omitempty"`   // URL of an image to use for the person who created this stream
	AuthorURL       string               `path:"authorUrl"      json:"authorUrl"       bson:"authorUrl,omitempty"`     // URL address of the person who created this stream
	Content         nebula.Container     `path:"content"        json:"content"         bson:"content,omitempty"`       // Content objects for this stream.
	Data            datatype.Map         `path:"data"           json:"data"            bson:"data"`                    // Set of data to populate into the Template.  This is validated by the JSON-Schema of the Template.
	Tags            []string             `path:"tags"           json:"tags"            bson:"tags"`                    // Organizational Tags
	ThumbnailImage  string               `path:"thumbnailImage" json:"thumbnailImage"  bson:"thumbnailImage"`          // Image to display next to the stream in lists.
	Rank            int                  `path:"rank"           json:"rank"            bson:"rank"`                    // If Template uses a custom sort order, then this is the value used to determine the position of this Stream.
	SourceID        primitive.ObjectID   `path:"sourceId"       json:"sourceId"        bson:"sourceId,omitempty"`      // Internal identifier of the source configuration that generated this stream
	SourceURL       string               `path:"sourceUrl"      json:"sourceUrl"       bson:"sourceUrl,omitempty"`     // URL of the original document published by the source server
	SourceUpdated   int64                `path:"sourceUpdated"  json:"sourceUpdated"   bson:"sourceUpdated,omitempty"` // Date the the source updated the original content.
	PublishDate     int64                `path:"publishDate"    json:"publishDate"     bson:"publishDate"`             // Unix timestamp of the date/time when this document is/was/will be first available on the domain.
	UnPublishDate   int64                `path:"unpublishDate " json:"unpublishDate"   bson:"unpublishDate"`           // Unix timestemp of the date/time when this document will no longer be available on the domain.
	journal.Journal `json:"journal" bson:"journal"`
}

// NewStream returns a fully initialized Stream object.
func NewStream() Stream {

	streamID := primitive.NewObjectID()

	return Stream{
		StreamID:  streamID,
		Token:     streamID.Hex(),
		ParentIDs: make([]primitive.ObjectID, 0),
		StateID:   "new",
		Criteria:  NewCriteria(),
		Tags:      make([]string, 0),
		Data:      make(datatype.Map),
		Content:   make(nebula.Container, 0),
	}
}

/*******************************************
 * DATA.OBJECT INTERFACE
 *******************************************/

// ID returns the primary key of this object
func (stream *Stream) ID() string {
	return stream.StreamID.Hex()
}

/*******************************************
 * OTHER DATA ACCESSORS
 *******************************************/

// GetContainer satisfies the content.Getter interface
func (stream *Stream) GetContainer() nebula.Container {
	if stream.Content == nil {
		stream.Content = nebula.Container{}
	}
	return stream.Content
}

// SetContainer satisfies the content.Setter interface
func (stream *Stream) SetContainer(value nebula.Container) {
	stream.Content = value
}

/*******************************************
 * OTHER METHODS
 *******************************************/

// HasParent returns TRUE if this Stream has a valid parentID
func (stream *Stream) HasParent() bool {
	return !stream.ParentID.IsZero()
}

// NewAttachment creates a new file Attachment linked to this Stream.
func (stream *Stream) NewAttachment(filename string) Attachment {
	result := NewAttachment(stream.StreamID)
	result.Original = filename

	return result
}

// Roles returns a list of all roles that match the provided authorization
func (stream *Stream) Roles(authorization *Authorization) []string {

	// Everyone has "public" access
	result := []string{"public"}

	if authorization == nil {
		return result
	}

	// Owners are hard-coded to do everything, so no other roles need to be returned.
	if authorization.DomainOwner {
		return []string{"owner"}
	}

	// Authors sometimes have special permissions, too.
	if !stream.AuthorID.IsZero() {
		if authorization.UserID == stream.AuthorID {
			result = append(result, "author")
		}
	}

	// Otherwise, append all roles matched from the criteria
	result = append(result, stream.Criteria.Roles(authorization.GroupIDs...)...)

	return result
}
