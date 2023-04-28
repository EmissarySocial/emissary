package model

import (
	"math"
	"time"

	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Stream corresponds to a top-level path on any Domain.
type Stream struct {
	StreamID        primitive.ObjectID           `json:"streamId"               bson:"_id"`                 // Unique identifier of this Stream.  (NOT USED PUBLICLY)
	ParentID        primitive.ObjectID           `json:"parentId"               bson:"parentId"`            // Unique identifier of the "parent" stream. (NOT USED PUBLICLY)
	ParentIDs       id.Slice                     `json:"parentIds"              bson:"parentIds"`           // List of all parent IDs, including the current parent.  This is used to generate "breadcrumbs" for the Stream.
	Rank            int                          `json:"rank"                   bson:"rank"`                // If Template uses a custom sort order, then this is the value used to determine the position of this Stream.
	Token           string                       `json:"token"                  bson:"token"`               // Unique value that identifies this element in the URL
	NavigationID    string                       `json:"navigationId"           bson:"navigationId"`        // Unique identifier of the "top-level" Stream that this record falls within. (NOT USED PUBLICLY)
	TemplateID      string                       `json:"templateId"             bson:"templateId"`          // Unique identifier (name) of the Template to use when rendering this Stream in HTML.
	SocialRole      string                       `json:"socialRole"             bson:"socialRole"`          // Role to use for this Stream in social integrations (Article, Note, Image, etc)
	StateID         string                       `json:"stateId"                bson:"stateId"`             // Unique identifier of the State this Stream is in.  This is used to populate the State information from the Template service at load time.
	Permissions     mapof.Object[sliceof.String] `json:"permissions"            bson:"permissions"`         // Permissions for which users can access this stream.
	DefaultAllow    id.Slice                     `json:"defaultAllow"           bson:"defaultAllow"`        // List of Groups that are allowed to perform the 'default' (view) action.  This is used to query general access to the Stream from the database, before performing server-based authentication.
	Document        DocumentLink                 `json:"document"               bson:"document"`            // Summary information (url, title, summary) for this Stream
	InReplyTo       DocumentLink                 `json:"inReplyTo,omitempty"    bson:"inReplyTo,omitempty"` // If this stream is a reply to another stream or web page, then this links to the original document.
	Content         Content                      `json:"content"                bson:"content,omitempty"`   // Body content object for this Stream.
	Widgets         set.Slice[StreamWidget]      `json:"widgets"                bson:"widgets"`             // Additional widgets to include when rendering this Stream.
	Data            mapof.Any                    `json:"data"                   bson:"data,omitempty"`      // Set of data to populate into the Template.  This is validated by the JSON-Schema of the Template.
	PublishDate     int64                        `json:"publishDate"            bson:"publishDate"`         // Unix timestamp of the date/time when this document is/was/will be first available on the domain.
	UnPublishDate   int64                        `json:"unpublishDate"          bson:"unpublishDate"`       // Unix timestemp of the date/time when this document will no longer be available on the domain.
	journal.Journal `json:"journal" bson:"journal"`
}

// NewStream returns a fully initialized Stream object.
func NewStream() Stream {

	streamID := primitive.NewObjectID()

	return Stream{
		StreamID:      streamID,
		Token:         streamID.Hex(),
		ParentID:      primitive.NilObjectID,
		ParentIDs:     id.NewSlice(),
		StateID:       "new",
		Permissions:   NewStreamPermissions(),
		InReplyTo:     NewDocumentLink(),
		Widgets:       NewStreamWidgets(),
		Data:          mapof.NewAny(),
		PublishDate:   math.MaxInt64,
		UnPublishDate: math.MaxInt64,
	}
}

// NewStreamPermissions returns a fully initialized Permissions object
func NewStreamPermissions() mapof.Object[sliceof.String] {
	return make(mapof.Object[sliceof.String])
}

// NewStreamWidgets returns a fully initialized StreamWidget slice
func NewStreamWidgets() set.Slice[StreamWidget] {
	return make(set.Slice[StreamWidget], 0)
}

/******************************************
 * data.Object Interface
 ******************************************/

// ID returns the primary key of this object
func (stream *Stream) ID() string {
	return stream.StreamID.Hex()
}

/******************************************
 * Other Data Accessors
 ******************************************/

func (stream *Stream) Permalink() string {
	return stream.Document.URL
}

func (stream *Stream) WidgetsByLocation(location string) []StreamWidget {

	return slice.Filter(stream.Widgets, func(widget StreamWidget) bool {
		return widget.Location == location
	})
}

func (stream *Stream) WidgetByID(streamWidgetID primitive.ObjectID) StreamWidget {

	for _, widget := range stream.Widgets {
		if widget.StreamWidgetID == streamWidgetID {
			return widget
		}
	}

	return StreamWidget{}
}

// GetSort returns the sortable value for this stream, based onthe provided fieldName
func (stream *Stream) GetSort(fieldName string) any {
	switch fieldName {
	case "publishDate":
		return stream.PublishDate
	case "document.label":
		return stream.Document.Label
	case "rank":
		return stream.Rank
	default:
		return 0
	}
}

/******************************************
 * RoleStateEnumerator Methods
 ******************************************/

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
	for _, author := range stream.Document.AttributedTo {
		if author.InternalID == authorization.UserID {
			result = append(result, MagicRoleAuthor)
		}
	}

	// If this Stream is in the current User's outbox, then they also have "self" permissions
	if stream.ParentID == authorization.UserID {
		result = append(result, MagicRoleMyself)
	}

	// Otherwise, append all roles matched from the permissions
	result = append(result, stream.PermissionRoles(authorization.AllGroupIDs()...)...)

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

/******************************************
 * Permission Methods
 ******************************************/

// AssignPermissions assigns a role to a group
func (stream *Stream) AssignPermission(role string, groupID primitive.ObjectID) {
	groupIDHex := groupID.Hex()

	if _, ok := stream.Permissions[groupIDHex]; !ok {
		stream.Permissions[groupIDHex] = []string{role}
		return
	}

	stream.Permissions[groupIDHex] = append(stream.Permissions[groupIDHex], role)
}

// PermissionGroups returns all groups that match the provided roles
func (stream *Stream) PermissionGroups(roles ...string) []primitive.ObjectID {

	result := make([]primitive.ObjectID, 0)

	for _, role := range roles {
		switch role {

		case "anonymous":
			result = append(result, MagicGroupIDAnonymous)

		case "authenticated":
			result = append(result, MagicGroupIDAuthenticated)

		}
	}

	for groupID, groupRoles := range stream.Permissions {
		if matchAny(roles, groupRoles) {
			if groupID, err := primitive.ObjectIDFromHex(groupID); err == nil {
				result = append(result, groupID)
			}
		}
	}

	return result
}

// PermissionRoles returns a unique list of all roles that the provided groups can access.
func (stream *Stream) PermissionRoles(groupIDs ...primitive.ObjectID) []string {

	result := []string{}

	// Copy values from group roles
	for _, groupID := range groupIDs {
		if roles, ok := stream.Permissions[groupID.Hex()]; ok {
			result = append(result, roles...)
		}
	}

	return result
}

// SimplePermissionModel returns a model object for displaying Simple Sharing.
func (stream *Stream) SimplePermissionModel() mapof.Any {

	// Special case if this is for EVERYBODY
	if _, ok := stream.Permissions[MagicGroupIDAnonymous.Hex()]; ok {
		return mapof.Any{
			"rule":     "anonymous",
			"groupIds": sliceof.NewString(),
		}
	}

	// Special case if this is for AUTHENTICATED
	if _, ok := stream.Permissions[MagicGroupIDAuthenticated.Hex()]; ok {
		return mapof.Any{
			"rule":     "authenticated",
			"groupIds": sliceof.NewString(),
		}
	}

	// Fall through means that additional groups are selected.
	// First, get all keys to the Groups map
	groupIDs := make(sliceof.String, len(stream.Permissions))
	index := 0

	for groupID := range stream.Permissions {
		groupIDs[index] = groupID
		index++
	}

	return mapof.Any{
		"rule":     "private",
		"groupIds": groupIDs,
	}
}

/******************************************
 * ActivityStreams Methods
 ******************************************/

// GetJSONLD returns a map document that conforms to the ActivityStreams 2.0 spec.
// This map will still need to be marshalled into JSON
func (stream Stream) GetJSONLD() mapof.Any {
	return mapof.Any{
		"@id":       stream.Document.URL,
		"@type":     stream.SocialRole,
		"id":        stream.Document.URL,
		"type":      stream.SocialRole,
		"url":       stream.Document.URL,
		"name":      stream.Document.Label,
		"summary":   stream.Document.Summary,
		"image":     stream.Document.ImageURL,
		"content":   stream.Content.HTML,
		"published": time.Unix(stream.PublishDate, 0).Format(time.RFC3339),
		"attributedTo": slice.Map(stream.Document.AttributedTo, func(person PersonLink) mapof.Any {
			return person.GetJSONLD()
		}),
	}
}

/******************************************
 * Other Methods
 ******************************************/

// HasParent returns TRUE if this Stream has a valid parentID
func (stream *Stream) HasParent() bool {
	return !stream.ParentID.IsZero()
}

// NewAttachment creates a new file Attachment linked to this Stream.
func (stream *Stream) NewAttachment(filename string) Attachment {
	result := NewAttachment(AttachmentTypeStream, stream.StreamID)
	result.Original = filename

	return result
}

func (stream *Stream) SetAttributedTo(people ...PersonLink) {
	stream.Document.AttributedTo = people
}

func (stream *Stream) AddAttributedTo(people ...PersonLink) {
	stream.Document.AttributedTo = append(stream.Document.AttributedTo, people...)
}
