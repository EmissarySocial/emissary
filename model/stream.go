package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/maps"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Stream corresponds to a top-level path on any Domain.
type Stream struct {
	StreamID        primitive.ObjectID   `path:"streamId"       json:"streamId"            bson:"_id"`                 // Unique identifier of this Stream.  (NOT USED PUBLICLY)
	ParentID        primitive.ObjectID   `path:"parentId"       json:"parentId"            bson:"parentId"`            // Unique identifier of the "parent" stream. (NOT USED PUBLICLY)
	ParentIDs       []primitive.ObjectID `path:"parentIds"      json:"parentIds"           bson:"parentIds"`           // Slice of all parent IDs of this Stream
	TemplateID      string               `path:"templateId"     json:"templateId"          bson:"templateId"`          // Unique identifier (name) of the Template to use when rendering this Stream in HTML.
	InReplyTo       string               `path:"inReplyTo"      json:"inReplyTo,omitempty" bson:"inReplyTo,omitempty"` // If this Stream is a reply to another Stream (or external document on the Interweb) then this field contains the ObjectID or URL of that other document.
	StateID         string               `path:"stateId"        json:"stateId"             bson:"stateId"`             // Unique identifier of the State this Stream is in.  This is used to populate the State information from the Template service at load time.
	DefaultAllow    []primitive.ObjectID `path:"defaultAllow"   json:"defaultAllow"        bson:"defaultAllow"`        // List of Groups that are allowed to perform the 'default' (view) action.  This is used to query general access to the Stream from the database, before performing server-based authentication.
	Permissions     Permissions          `path:"permissions"    json:"permissions"         bson:"permissions"`         // Permissions for which users can access this stream.
	Token           string               `path:"token"          json:"token"               bson:"token"`               // Unique value that identifies this element in the URL
	Label           string               `path:"label"          json:"label"               bson:"label"`               // Text to display in lists of streams, probably displayed at top of stream page, too.
	Description     string               `path:"description"    json:"description"         bson:"description"`         // Brief summary of this stream, used in lists of streams
	Author          AuthorLink           `path:"author"         json:"author"              bson:"author"`              // Author information for this stream
	Origin          OriginLink           `path:"origin"         json:"origin,omitempty"    bson:"origin"`              // If imported, the external document where this stream came from
	ReplyTo         ReplyToLink          `path:"replyTo"        json:"replyTo,omitempty"   bson:"replyTo,omitempty"`   // If this stream is a reply to another stream or web page, then this links to the original document.
	Content         Content              `path:"content"        json:"content"             bson:"content,omitempty"`   // Content objects for this stream.
	Data            maps.Map             `path:"data"           json:"data"                bson:"data,omitempty"`      // Set of data to populate into the Template.  This is validated by the JSON-Schema of the Template.
	Tags            []string             `path:"tags"           json:"tags"                bson:"tags,omitempty"`      // Organizational Tags
	ThumbnailImage  string               `path:"thumbnailImage" json:"thumbnailImage"      bson:"thumbnailImage"`      // Image to display next to the stream in lists.
	Rank            int                  `path:"rank"           json:"rank"                bson:"rank"`                // If Template uses a custom sort order, then this is the value used to determine the position of this Stream.
	AsFeature       bool                 `path:"asFeature"      json:"asFeature"           bson:"asFeature"`           // If TRUE, then this stream is a "feature" that is meant to be embedded into other stream views.
	PublishDate     int64                `path:"publishDate"    json:"publishDate"         bson:"publishDate"`         // Unix timestamp of the date/time when this document is/was/will be first available on the domain.
	UnPublishDate   int64                `path:"unpublishDate " json:"unpublishDate"       bson:"unpublishDate"`       // Unix timestemp of the date/time when this document will no longer be available on the domain.
	journal.Journal `path:"journal" json:"journal" bson:"journal"`
}

// NewStream returns a fully initialized Stream object.
func NewStream() Stream {

	streamID := primitive.NewObjectID()

	return Stream{
		StreamID:    streamID,
		Token:       streamID.Hex(),
		ParentID:    primitive.NilObjectID,
		ParentIDs:   make([]primitive.ObjectID, 0),
		StateID:     "new",
		Permissions: NewPermissions(),
		Tags:        make([]string, 0),
		Data:        make(maps.Map),
	}
}

/*******************************************
 * data.Object Interface
 *******************************************/

// ID returns the primary key of this object
func (stream *Stream) ID() string {
	return stream.StreamID.Hex()
}

/*******************************************
 * Other Data Accessors
 *******************************************/

// Links returns all resources linked to this Stream.  Some links may be empty.
func (stream *Stream) Links() []Link {
	return []Link{
		stream.Author.Link(),
		stream.Origin.Link(),
		stream.ReplyTo.Link(),
	}
}

// SetAuthor populates the denormalized author information into this stream.
func (stream *Stream) SetAuthor(user *User) {
	stream.Author = user.AuthorLink()
}

/*******************************************
 * RoleStateEnumerator Methods
 *******************************************/

// State returns the current state of this Stream.  It is
// part of the implementation of the RoleStateEmulator interface
func (stream *Stream) State() string {
	return stream.StateID
}

// Roles returns a list of all roles that match the provided authorization
func (stream *Stream) Roles(authorization *Authorization) []string {

	// Everyone has "anonymous" access
	result := []string{MagicRoleAnonymous}

	if authorization == nil {
		return result
	}

	// Owners are hard-coded to do everything, so no other roles need to be returned.
	if authorization.DomainOwner {
		return []string{MagicRoleOwner}
	}

	if authorization.IsAuthenticated() {
		result = append(result, MagicRoleAuthenticated)
	}

	// Authors sometimes have special permissions, too.
	if !stream.Author.InternalID.IsZero() {
		if authorization.UserID == stream.Author.InternalID {
			result = append(result, MagicRoleAuthor)
		}
	}

	// Otherwise, append all roles matched from the permissions
	result = append(result, stream.Permissions.Roles(authorization.AllGroupIDs()...)...)

	return result
}

// DefaultAllowAnonymous returns TRUE if a Stream's default action (VIEW)
// is visible to anonymous visitors
func (stream *Stream) DefaultAllowAnonymous() bool {
	for index := range stream.DefaultAllow {
		if stream.DefaultAllow[index] == MagicGroupIDAnonymous {
			return true
		}
	}
	return false
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
