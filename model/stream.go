package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/derp"
	"github.com/benpate/path"
	"github.com/benpate/slice"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Stream corresponds to a top-level path on any Domain.
type Stream struct {
	StreamID       primitive.ObjectID     `json:"streamId"        bson:"_id"`                 // Unique identifier of this Stream.  (NOT USED PUBLICLY)
	ParentID       primitive.ObjectID     `json:"parentId"        bson:"parentId"`            // Unique identifier of the "parent" stream. (NOT USED PUBLICLY)
	FolderID       primitive.ObjectID     `json:"folderId"        bson:"folderId"`            // Unique identifier of the "folder" where this stream is stored (NOT USED PUBLICLY)
	TemplateID     string                 `json:"templateId"      bson:"templateId"`          // Unique identifier (name) of the Template to use when rendering this Stream in HTML.
	StateID        string                 `json:"stateId"         bson:"stateId"`             // Unique identifier of the State this Stream is in.  This is used to populate the State information from the Template service at load time.
	GroupRoles     map[string][]string    `json:"groupRoles"      bson:"groupRoles"`          // Map of Role names to the one or more Group names that can perform that role.
	Token          string                 `json:"token"           bson:"token"`               // Unique value that identifies this element in the URL
	URL            string                 `json:"url"             bson:"url"`                 // Unique URL of this Stream.  This duplicates the "token" field a bit, but it (hopefully?) makes access easier.
	Label          string                 `json:"label"           bson:"label"`               // Text to display in lists of streams, probably displayed at top of stream page, too.
	Description    string                 `json:"description"     bson:"description"`         // Brief summary of this stream, used in lists of streams
	ThumbnailImage string                 `json:"thumbnailImage"  bson:"thumbnailImage"`      // Image to display next to the stream in lists.
	AuthorID       primitive.ObjectID     `json:"authorId"        bson:"authorId"`            // Unique identifier of the person who created this stream (NOT USED PUBLICLY)
	AuthorName     string                 `json:"authorName"      bson:"authorName"`          // Full name of the person who created this stream
	AuthorImage    string                 `json:"authorImage"     bson:"authorImage"`         // URL of an image to use for the person who created this stream
	AuthorURL      string                 `json:"authorURL"       bson:"authorURL"`           // URL address of the person who created this stream
	Tags           []string               `json:"tags"            bson:"tags"`                // Organizational Tags
	Data           map[string]interface{} `json:"data"            bson:"data"`                // Set of data to populate into the Template.  This is validated by the JSON-Schema of the Template.
	Source         StreamSourceType       `json:"source"          bson:"source,omitempty"`    // Identifies the remote source
	SourceID       primitive.ObjectID     `json:"sourceId"        bson:"sourceId,omitempty"`  // Internal identifier of the source configuration that generated this stream
	SourceURL      string                 `json:"sourceURL"       bson:"sourceURL,omitempty"` // URL of the original document published by the source server
	PublishDate    int64                  `json:"publishDate"     bson:"publishDate"`         // Unix timestamp of the date/time when this document is/was/will be first available on the domain.
	UnPublishDate  int64                  `json:"unpublishDate"   bson:"unpublishDate"`       // Unix timestemp of the date/time when this document will no longer be available on the domain.

	journal.Journal `json:"journal" bson:"journal"`
}

// NewStream returns a fully initialized Stream object.
func NewStream() Stream {

	return Stream{
		StreamID:   primitive.NewObjectID(),
		StateID:    "new",
		GroupRoles: make(map[string][]string, 0),
		Tags:       []string{},
		Data:       map[string]interface{}{},
	}
}

// ID returns the primary key of this object
func (stream Stream) ID() string {
	return stream.StreamID.Hex()
}

// HasParent returns TRUE if this Stream has a valid parentID
func (stream Stream) HasParent() bool {
	return !stream.ParentID.IsZero()
}

/*
// MatchAnonymous returns TRUE if this Stream does not
// require any access permissions
func (stream Stream) MatchAnonymous() bool {
	return stream.State.MatchAnonymous()
}

// MatchRoles returns TRUE if this Stream is accessible
// by a user with the provided roles .
func (stream Stream) MatchRoles(roles ...string) bool {
	return stream.State.MatchRoles(roles...)
}

// MatchGroups returns TRUE if
func (stream Stream) MatchGroups(view string, groups ...string) bool {

	roles := stream.Roles(groups...)

	if stream.State.MatchRoles(roles...) {
		if view, ok := stream.View(view); ok {
			if view.MatchRoles(roles...) {
				return true
			}
		}
	}

	return false
}

// View searches for the first view in this stream that matches the provided ID.
// If found, the view is returned along with a TRUE.
// If no view matches, and empty view is returned along with a FALSE.
func (stream Stream) View(viewID string) (View, bool) {
	return stream.State.View(viewID)
}

// Transition returns the transition matching the provided ID, and TRUE.
// If no transition matches, then an empty transition is returned along with a FALSE.
func (stream Stream) Transition(transitionID string) (Transition, bool) {
	return stream.State.Transition(transitionID)
}
*/

// Roles returns a unique list of all roles that the provided groups are granted
// in this stream
func (stream Stream) Roles(groups ...string) []string {

	result := []string{}

	for _, group := range groups {
		if roles, ok := stream.GroupRoles[group]; ok {
			result = slice.AddUnique(result, roles...)
		}
	}

	return result
}

// GetPath implements the path.Getter interface.  It looks up
// data within this Stream and returns it to the caller.
func (stream Stream) GetPath(p path.Path) (interface{}, error) {

	switch p.Head() {

	case "data":
		return p.Tail().Get(stream.Data)

	case "label":
		return stream.Label, nil

	case "description":
		return stream.Description, nil

	case "thumbnailImage":
		return stream.ThumbnailImage, nil
	}

	return nil, derp.New(500, "ghost.model.Stream", "Unrecognized path", p)
}

// SetPath implements the path.Setter interface.  It takes any data value
// and tries to set it to the correct path within this Stream.
func (stream *Stream) SetPath(p path.Path, value interface{}) error {

	var property *string

	// Properties that can be set
	switch p.Head() {

	case "data":
		return p.Tail().Set(stream.Data, value)

	case "label":
		if p.IsTailEmpty() {
			property = &stream.Label
		}

	case "description":
		if p.IsTailEmpty() {
			property = &stream.Description
		}

	case "thumbnailImage":
		if p.IsTailEmpty() {
			property = &stream.ThumbnailImage
		}
	}

	// Set property (if it is valid)
	if property != nil {
		if v, ok := value.(string); ok {
			*property = v
			return nil
		}

		return derp.New(500, "ghost.model.Stream.SetPath", "Label must be a string", value)
	}

	// Fall through means failure.  Own it.
	return derp.New(500, "ghost.model.Stream", "Unrecognized path", p)
}
