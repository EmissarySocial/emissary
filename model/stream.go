package model

import (
	"math"
	"time"

	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/data/journal"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/toot/object"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Stream corresponds to a top-level path on any Domain.
type Stream struct {
	StreamID        primitive.ObjectID           `json:"streamId"               bson:"_id"`                    // Unique identifier of this Stream.  (NOT USED PUBLICLY)
	ParentID        primitive.ObjectID           `json:"parentId"               bson:"parentId"`               // Unique identifier of the "parent" stream. (NOT USED PUBLICLY)
	ParentIDs       id.Slice                     `json:"parentIds"              bson:"parentIds"`              // List of all parent IDs, including the current parent.  This is used to generate "breadcrumbs" for the Stream.
	Rank            int                          `json:"rank"                   bson:"rank"`                   // If Template uses a custom sort order, then this is the value used to determine the position of this Stream.
	NavigationID    string                       `json:"navigationId"           bson:"navigationId"`           // Unique identifier of the "top-level" Stream that this record falls within. (NOT USED PUBLICLY)
	TemplateID      string                       `json:"templateId"             bson:"templateId"`             // Unique identifier (name) of the Template to use when rendering this Stream in HTML.
	StateID         string                       `json:"stateId"                bson:"stateId"`                // Unique identifier of the State this Stream is in.  This is used to populate the State information from the Template service at load time.
	SocialRole      string                       `json:"socialRole,omitempty"   bson:"socialRole,omitempty"`   // Role to use for this Stream in social integrations (Article, Note, Image, etc)
	Permissions     mapof.Object[sliceof.String] `json:"permissions,omitempty"  bson:"permissions,omitempty"`  // Permissions for which users can access this stream.
	DefaultAllow    id.Slice                     `json:"defaultAllow,omitempty" bson:"defaultAllow,omitempty"` // List of Groups that are allowed to perform the 'default' (view) action.  This is used to query general access to the Stream from the database, before performing server-based authentication.
	URL             string                       `json:"url,omitempty"          bson:"url,omitempty"`          // URL of the original document
	Token           string                       `json:"token,omitempty"        bson:"token,omitempty"`        // Unique value that identifies this element in the URL
	Label           string                       `json:"label,omitempty"        bson:"label,omitempty"`        // Label/Title of the document
	Summary         string                       `json:"summary,omitempty"      bson:"summary,omitempty"`      // Brief summary of the document
	ImageURL        string                       `json:"imageUrl,omitempty"     bson:"imageUrl,omitempty"`     // URL of the cover image for this document's image
	Content         Content                      `json:"content,omitempty"      bson:"content,omitempty"`      // Body content object for this Stream.
	Widgets         set.Slice[StreamWidget]      `json:"widgets,omitempty"      bson:"widgets,omitempty"`      // Additional widgets to include when rendering this Stream.
	Tags            sliceof.Object[Tag]          `json:"tags,omitempty"         bson:"tags,omitempty"`         // List of tags that are associated with this document
	Data            mapof.Any                    `json:"data,omitempty"         bson:"data,omitempty"`         // Set of data to populate into the Template.  This is validated by the JSON-Schema of the Template.
	AttributedTo    PersonLink                   `json:"attributedTo,omitempty" bson:"attributedTo,omitempty"` // List of people who are attributed to this document
	Context         string                       `json:"context,omitempty"      bson:"context,omitempty"`      // Context of this document (usually a URL)
	InReplyTo       string                       `json:"inReplyTo,omitempty"    bson:"inReplyTo"`              // If this stream is a reply to another stream or web page, then this links to the original document.
	PublishDate     int64                        `json:"publishDate"            bson:"publishDate"`            // Unix timestamp of the date/time when this document is/was/will be first available on the domain.
	UnPublishDate   int64                        `json:"unpublishDate"          bson:"unpublishDate"`          // Unix timestemp of the date/time when this document will no longer be available on the domain.
	journal.Journal `json:"journal" bson:",inline"`
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
		Widgets:       NewStreamWidgets(),
		Data:          mapof.NewAny(),
		Tags:          sliceof.NewObject[Tag](),
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
	return stream.URL
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
	case "label":
		return stream.Label
	case "rank":
		return stream.Rank
	default:
		return 0
	}
}

/******************************************
 * StateSetter Methods
 ******************************************/

func (stream *Stream) SetState(stateID string) {
	stream.StateID = stateID
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

	if authorization.IsAuthenticated() {

		// Owners are hard-coded to do everything, so no other roles need to be returned.
		if authorization.DomainOwner {
			return []string{MagicRoleOwner}
		}

		result = append(result, MagicRoleAuthenticated)

		// Authors sometimes have special permissions, too.
		if stream.AttributedTo.UserID == authorization.UserID {
			result = append(result, MagicRoleAuthor)
		}

		// If this Stream is in the current User's outbox, then they also have "self" permissions
		if stream.ParentID == authorization.UserID {
			result = append(result, MagicRoleMyself)
		}
	}

	// Otherwise, append all roles matched from the permissions
	result = append(result, stream.PermissionRoles(authorization.AllGroupIDs()...)...)

	return result
}

// DefaultAllowAnonymous returns TRUE if a Stream's default action (VIEW)
// is visible to anonymous visitors
func (stream *Stream) DefaultAllowAnonymous() bool {
	for _, group := range stream.DefaultAllow {
		if group == MagicGroupIDAnonymous {
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

func (stream Stream) ActivityPubURL() string {
	return stream.URL
}

func (stream Stream) ActivityPubType() string {
	return stream.SocialRole
}

func (stream Stream) ActivityPubLikesURL() string {
	return stream.URL + "/pub/likes"
}

func (stream Stream) ActivityPubDislikesURL() string {
	return stream.URL + "/pub/dislikes"
}

func (stream Stream) ActivityPubSharesURL() string {
	return stream.URL + "/pub/shares"
}

func (stream Stream) ActivityPubRepliesURL() string {
	return stream.URL + "/pub/replies"
}

func (stream Stream) ActivityPubResponses(responseType string) string {
	switch responseType {

	case ResponseTypeLike:
		return stream.ActivityPubLikesURL()

	case ResponseTypeDislike:
		return stream.ActivityPubDislikesURL()
	}

	return stream.ActivityPubSharesURL()
}

/******************************************
 * Publishing MetaData
 ******************************************/

// IsPublished rturns TRUE if this Stream is currently published
func (stream *Stream) IsPublished() bool {
	now := time.Now().Unix()
	return (stream.PublishDate < now) && (stream.UnPublishDate > now)
}

// PublishActivity returns the ActivityType that should be used when publishing this Stream (either Create or Update)
func (stream *Stream) PublishActivity() string {
	if stream.IsPublished() {
		return vocab.ActivityTypeUpdate
	}

	return vocab.ActivityTypeCreate
}

/******************************************
 * Mastodon API Methods
 ******************************************/

func (stream Stream) Toot() object.Status {

	return object.Status{
		ID:          stream.StreamID.Hex(),
		URI:         stream.ActivityPubURL(),
		CreatedAt:   time.Unix(stream.PublishDate, 0).Format(time.RFC3339),
		Account:     stream.AttributedTo.Toot(),
		Content:     stream.Content.HTML,
		Visibility:  "public",
		SpoilerText: stream.Label,
		URL:         stream.URL,
		InReplyToID: stream.InReplyTo,
	}
}

func (stream Stream) GetRank() int64 {
	return int64(stream.Rank)
}

/******************************************
 * Other Methods
 ******************************************/

func (stream *Stream) DocumentLink() DocumentLink {
	return DocumentLink{
		ID:    stream.StreamID,
		URL:   stream.URL,
		Label: stream.Label,
	}
}

// HasParent returns TRUE if this Stream has a valid parentID
func (stream *Stream) HasParent() bool {
	return !stream.ParentID.IsZero()
}

// SetAttributedTo sets the list of people that this Stream is attributed to
func (stream *Stream) SetAttributedTo(person PersonLink) {
	stream.AttributedTo = person
}
