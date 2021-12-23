package model

import (
	"github.com/benpate/convert"
	"github.com/benpate/data/journal"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/content"
	"github.com/benpate/path"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Stream corresponds to a top-level path on any Domain.
type Stream struct {
	StreamID        primitive.ObjectID   `json:"streamId"        bson:"_id"`                     // Unique identifier of this Stream.  (NOT USED PUBLICLY)
	ParentID        primitive.ObjectID   `json:"parentId"        bson:"parentId"`                // Unique identifier of the "parent" stream. (NOT USED PUBLICLY)
	ParentIDs       []primitive.ObjectID `json:"parentIds"       bson:"parentIds"`               // Slice of all parent IDs of this Stream
	TemplateID      string               `json:"templateId"      bson:"templateId"`              // Unique identifier (name) of the Template to use when rendering this Stream in HTML.
	StateID         string               `json:"stateId"         bson:"stateId"`                 // Unique identifier of the State this Stream is in.  This is used to populate the State information from the Template service at load time.
	Criteria        Criteria             `json:"criteria"        bson:"criteria"`                // Criteria for which users can access this stream.
	Token           string               `json:"token"           bson:"token"`                   // Unique value that identifies this element in the URL
	Label           string               `json:"label"           bson:"label,omitempty"`         // Text to display in lists of streams, probably displayed at top of stream page, too.
	Description     string               `json:"description"     bson:"description,omitempty"`   // Brief summary of this stream, used in lists of streams
	AuthorID        primitive.ObjectID   `json:"authorId"        bson:"authorId,omitempty"`      // Unique identifier of the person who created this stream (NOT USED PUBLICLY)
	AuthorName      string               `json:"authorName"      bson:"authorName,omitempty"`    // Full name of the person who created this stream
	AuthorImage     string               `json:"authorImage"     bson:"authorImage,omitempty"`   // URL of an image to use for the person who created this stream
	AuthorURL       string               `json:"authorURL"       bson:"authorURL,omitempty"`     // URL address of the person who created this stream
	Content         content.Content      `json:"content"         bson:"content,omitempty"`       // Content objects for this stream.
	Data            datatype.Map         `json:"data"            bson:"data,omitempty"`          // Set of data to populate into the Template.  This is validated by the JSON-Schema of the Template.
	Tags            []string             `json:"tags"            bson:"tags,omitempty"`          // Organizational Tags
	ThumbnailImage  string               `json:"thumbnailImage"  bson:"thumbnailImage"`          // Image to display next to the stream in lists.
	Rank            int                  `json:"rank"            bson:"rank"`                    // If Template uses a custom sort order, then this is the value used to determine the position of this Stream.
	SourceID        primitive.ObjectID   `json:"sourceId"        bson:"sourceId,omitempty"`      // Internal identifier of the source configuration that generated this stream
	SourceURL       string               `json:"sourceUrl"       bson:"sourceUrl,omitempty"`     // URL of the original document published by the source server
	SourceUpdated   int64                `json:"sourceUpdated"   bson:"sourceUpdated,omitempty"` // Date the the source updated the original content.
	PublishDate     int64                `json:"publishDate"     bson:"publishDate"`             // Unix timestamp of the date/time when this document is/was/will be first available on the domain.
	UnPublishDate   int64                `json:"unpublishDate"   bson:"unpublishDate"`           // Unix timestemp of the date/time when this document will no longer be available on the domain.
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
		Content:   make(content.Content, 0),
	}
}

/*******************************************
 * DATA.OBJECT INTERFACE
 *******************************************/

// ID returns the primary key of this object
func (stream *Stream) ID() string {
	return stream.StreamID.Hex()
}

// GetPath implements the path.Getter interface.  It looks up
// data within this Stream and returns it to the caller.
func (stream *Stream) GetPath(p path.Path) (interface{}, error) {

	if p.IsEmpty() {
		return nil, derp.New(500, "ghost.model.Stream", "Unrecognized path", p)
	}

	property := p.Head()

	// Properties that can be retrieved
	switch property {

	case "label":
		return stream.Label, nil

	case "description":
		return stream.Description, nil

	case "thumbnailImage":
		return stream.ThumbnailImage, nil

	case "criteria":
		return stream.Criteria.GetPath(p.Tail())

	case "rank":
		return stream.Rank, nil

	default:
		return stream.Data[property], nil
	}
}

// SetPath implements the path.Setter interface.  It takes any data value
// and tries to set it to the correct path within this Stream.
func (stream *Stream) SetPath(p path.Path, value interface{}) error {

	if p.IsEmpty() {
		return derp.New(500, "ghost.model.Stream", "Unrecognized path", p)
	}

	property := p.Head()

	// Properties that can be set
	switch property {

	case "label":
		stream.Label = convert.String(value)

	case "description":
		stream.Description = convert.String(value)

	case "thumbnailImage":
		stream.ThumbnailImage = convert.String(value)

	case "criteria":
		return stream.Criteria.SetPath(p.Tail(), value)

	case "rank":
		stream.Rank = convert.Int(value)

	default:
		return stream.Data.SetPath(p, value)
	}

	return nil
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

	result := make([]string, 0)

	if stream.Criteria.Public {
		result = append(result, "public")
	}

	if authorization == nil {
		return result
	}

	if !stream.AuthorID.IsZero() {
		if authorization.UserID == stream.AuthorID {
			result = append(result, "author")
		}
	}

	if authorization.DomainOwner {
		result = append(result, "owner")
	}

	result = append(result, stream.Criteria.Roles(authorization.GroupIDs...)...)

	return result
}
