package model

import (
	"math"
	"net/url"
	"time"

	"github.com/EmissarySocial/emissary/tools/datetime"
	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/data/journal"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/delta"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/toot/object"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Stream corresponds to a top-level path on any Domain.
type Stream struct {
	StreamID         primitive.ObjectID      `bson:"_id"`                    // Unique identifier of this Stream.
	ParentID         primitive.ObjectID      `bson:"parentId"`               // Unique identifier of the "parent" stream.
	ParentIDs        id.Slice                `bson:"parentIds"`              // List of all parent IDs, including the current parent.  This is used to generate "breadcrumbs" for the Stream.
	Rank             int                     `bson:"rank"`                   // If Template uses a custom sort order, then this is the value used to determine the position of this Stream.
	RankAlt          int                     `bson:"rankAlt"`                // Alternate sort criteria
	NavigationID     string                  `bson:"navigationId"`           // Unique identifier of the "top-level" Stream that this record falls within.
	TemplateID       string                  `bson:"templateId"`             // Unique identifier (name) of the Template to use when building this Stream in HTML.
	ParentTemplateID string                  `bson:"parentTemplateId"`       // Unique identifier (name) of the parent's Template.
	StateID          string                  `bson:"stateId"`                // Unique identifier of the State this Stream is in.  This is used to populate the State information from the Template service at load time.
	SocialRole       string                  `bson:"socialRole,omitempty"`   // Role to use for this Stream in social integrations (Article, Note, Image, etc)
	Groups           mapof.Object[id.Slice]  `bson:"groups,omitempty"`       // Groups maps roles into GroupIDs for this Stream.  This is used to determine access rights for the Stream.
	Circles          mapof.Object[id.Slice]  `bson:"circles,omitempty"`      // Circles maps roles into CircleIDs for this Stream.  This is used to determine access rights for the Stream.
	Products         mapof.Object[id.Slice]  `bson:"products,omitempty"`     // Products maps roles into ProductIDs for this Stream.  This is used to determine access rights for the Stream.
	PrivilegeIDs     Permissions             `bson:"privilegeIds,omitempty"` // List of ALL Privilege IDs that grant ANY permissions to this Stream (denormalized from the Products and Circles maps)
	DefaultAllow     Permissions             `bson:"defaultAllow,omitempty"` // List of Groups that are allowed to perform the 'default' (view) action.  This is used to query general access to the Stream from the database, before performing server-based authentication.
	URL              string                  `bson:"url,omitempty"`          // URL of the original document
	Token            string                  `bson:"token,omitempty"`        // Unique value that identifies this element in the URL
	Label            string                  `bson:"label,omitempty"`        // Label/Title of the document
	Summary          string                  `bson:"summary,omitempty"`      // Brief summary of the document
	Icon             string                  `bson:"icon,omitempty"`         // Icon CSS/Token for the document
	IconURL          string                  `bson:"iconUrl,omitempty"`      // URL of this document's icon/thumbnail image
	Context          string                  `bson:"context,omitempty"`      // Context of this document (usually a URL)
	InReplyTo        string                  `bson:"inReplyTo"`              // If this stream is a reply to another stream or web page, then this links to the original document.
	AttributedTo     PersonLink              `bson:"attributedTo,omitempty"` // List of people who are attributed to this document
	Content          Content                 `bson:"content,omitempty"`      // Body content object for this Stream.
	Widgets          set.Slice[StreamWidget] `bson:"widgets,omitempty"`      // Additional widgets to include when building this Stream.
	Hashtags         sliceof.String          `bson:"hashtags,omitempty"`     // List of hashtags that are associated with this document
	Places           sliceof.Object[Place]   `bson:"places,omitempty"`       // List of locations that are associated with this document
	Data             mapof.Any               `bson:"data,omitempty"`         // Set of data to populate into the Template.  This is validated by the JSON-Schema of the Template.
	StartDate        datetime.DateTime       `bson:"startDate,omitempty"`    // Date/Time to publish as a "start date" for this Stream (semantics are dependent on the Template)
	EndDate          datetime.DateTime       `bson:"endDate,omitempty"`      // Date/Time to publish as an "end date" for this Stream (semantics are dependent on the Template)
	Syndication      delta.Slice[string]     `bson:"syndication,omitempty"`  // List of external services that this Stream has been syndicated to.
	Shuffle          int64                   `bson:"shuffle"`                // Random number used to shuffle the order of Streams in a list.
	PublishDate      int64                   `bson:"publishDate"`            // Unix timestamp of the date/time when this document is/was/will be first available on the domain.
	UnPublishDate    int64                   `bson:"unpublishDate"`          // Unix timestemp of the date/time when this document will no longer be available on the domain.
	IsFeatured       bool                    `bson:"isFeatured"`             // TRUE if this Stream is featured by its parent container.
	IsSubscribable   bool                    `bson:"isSubscribable"`         // TRUE if this Stream uses the Products service to determine access rights.

	// Deprecated: Permissions maps UserIDs/GroupIDs into Roles for this Stream.
	// Permissions mapof.Object[sliceof.String] `bson:"permissions,omitempty"`

	journal.Journal `bson:",inline"`
}

// NewStream returns a fully initialized Stream object.
func NewStream() Stream {

	streamID := primitive.NewObjectID()

	return Stream{
		StreamID:      streamID,
		Token:         streamID.Hex(),
		ParentID:      primitive.NilObjectID,
		ParentIDs:     id.NewSlice(),
		StateID:       "default",
		Groups:        mapof.NewObject[id.Slice](),
		Circles:       mapof.NewObject[id.Slice](),
		Products:      mapof.NewObject[id.Slice](),
		DefaultAllow:  NewPermissions(),
		PrivilegeIDs:  NewPermissions(),
		Widgets:       NewStreamWidgets(),
		Data:          mapof.NewAny(),
		Hashtags:      sliceof.NewString(),
		PublishDate:   math.MaxInt64,
		UnPublishDate: math.MaxInt64,
		Syndication:   delta.NewSlice[string](),
	}
}

// NewStreamWidgets returns a fully initialized StreamWidget slice
func NewStreamWidgets() set.Slice[StreamWidget] {
	return make(set.Slice[StreamWidget], 0)
}

/******************************************
 * data.Object Interface
 ******************************************/

// ID returns the primary key of this object
func (stream Stream) ID() string {
	return stream.StreamID.Hex()
}

/******************************************
 * Other Data Accessors
 ******************************************/

func (stream Stream) Permalink() string {
	return stream.URL
}

// SummaryOrContent returns the stream summary -- if it exists -- otherwise, the content HTML.
func (stream Stream) SummaryOrContent() string {
	if stream.Summary != "" {
		return stream.Summary
	}

	return stream.Content.HTML
}

func (stream Stream) WidgetsByLocation(location string) []StreamWidget {

	return slice.Filter(stream.Widgets, func(widget StreamWidget) bool {
		return widget.Location == location
	})
}

func (stream Stream) WidgetByID(streamWidgetID primitive.ObjectID) StreamWidget {

	for _, widget := range stream.Widgets {
		if widget.StreamWidgetID == streamWidgetID {
			return widget
		}
	}

	return StreamWidget{}
}

// GetSort returns the sortable value for this stream, based onthe provided fieldName
func (stream Stream) GetSort(fieldName string) any {
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
 * StateSetter Interface
 ******************************************/

func (stream *Stream) SetState(stateID string) {
	stream.StateID = stateID
}

/******************************************
 * AccessLister Interface
 ******************************************/

// State returns the current state of this Stream.
// It is part of the AccessLister interface
func (stream Stream) State() string {
	return stream.StateID
}

// IsAuthor returns TRUE if the provided UserID the author of this Stream
// It is part of the AccessLister interface
func (stream Stream) IsAuthor(authorID primitive.ObjectID) bool {
	return !authorID.IsZero() && authorID == stream.AttributedTo.UserID
}

// IsMyself returns TRUE if this object directly represents the provided UserID
// It is part of the AccessLister interface
func (stream Stream) IsMyself(_ primitive.ObjectID) bool {
	return false
}

// RolesToGroupIDs returns a slice of Group IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (stream Stream) RolesToGroupIDs(roles ...string) Permissions {

	result := NewPermissions()

	for _, role := range roles {

		switch role {

		case MagicRoleAnonymous:
			return NewAnonymousPermissions()

		case MagicRoleAuthenticated:
			return NewAuthenticatedPermissions()

		case MagicRoleAuthor:
			result = append(result, stream.AttributedTo.UserID)

		default:
			result = append(result, stream.Groups[role]...)
		}
	}

	return result
}

// RolesToPrivilegeIDs returns a slice of Privileges that grant any of the requested roles
// It is part of the AccessLister interface
func (stream Stream) RolesToPrivilegeIDs(roles ...string) Permissions {

	result := NewPermissions()

	for _, role := range roles {
		if circles, exists := stream.Circles[role]; exists {
			result = append(result, circles...)
		}

		if products, exists := stream.Products[role]; exists {
			result = append(result, products...)
		}
	}

	return result
}

/******************************************
 * Permission Methods
 ******************************************/

// DefaultAllowAnonymous returns TRUE if a Stream's default action (VIEW)
// is visible to anonymous visitors
func (stream Stream) DefaultAllowAnonymous() bool {
	for _, group := range stream.DefaultAllow {
		if group == MagicGroupIDAnonymous {
			return true
		}
	}
	return false
}

/******************************************
 * Privilege Methods
 ******************************************/

func (stream Stream) IsPublic() bool {
	return stream.DefaultAllowAnonymous()
}

// HasPrivileges returns TRUE if this Stream includes special permissions for any Privilege
func (stream Stream) HasPrivileges() bool {
	return (len(stream.Circles) > 0) || (len(stream.Products) > 0)
}

// ProductIDs returns a slice of ALL Product IDs that are associated with this Stream,
// regardless of the role(s) that they participate in.
func (stream Stream) ProductIDs() id.Slice {
	return flatten(stream.Products)
}

func (stream Stream) GroupIDs() id.Slice {
	return flatten(stream.Groups)
}

func (stream Stream) CircleIDs() id.Slice {
	return flatten(stream.Circles)
}

/******************************************
 * ActivityStream Methods
 ******************************************/

func (stream Stream) ActivityPubURL() string {
	return stream.URL
}

func (stream Stream) ActivityPubInboxURL() string {
	return stream.URL + "/pub/inbox"
}

func (stream Stream) ActivityPubOutboxURL() string {
	return stream.URL + "/pub/outbox"
}

func (stream Stream) ActivityPubFollowersURL() string {
	return stream.URL + "/pub/followers"
}

func (stream Stream) ActivityPubAnnouncedURL() string {
	return stream.URL + "/pub/shared"
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

func (stream Stream) ActivityPubChildrenURL() string {
	return stream.URL + "/pub/children"
}

func (stream Stream) ActivityPubResponses(responseType string) string {

	switch responseType {

	case vocab.ActivityTypeLike:
		return stream.ActivityPubLikesURL()

	case vocab.ActivityTypeDislike:
		return stream.ActivityPubDislikesURL()
	}

	return stream.ActivityPubSharesURL()
}

/******************************************
 * Publishing MetaData
 ******************************************/

// IsPublished returns TRUE if this Stream is currently published
func (stream Stream) IsPublished() bool {

	// RULE: Deleted streams are not published
	if stream.DeleteDate > 0 {
		return false
	}

	// Otherwise, check that "now" is in between the Publish and UnPublish dates
	now := time.Now().Unix()
	return (stream.PublishDate <= now) && (stream.UnPublishDate > now)
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

// HasParent returns TRUE if this Stream has a valid parentID
func (stream Stream) HasParent() bool {
	return !stream.ParentID.IsZero()
}

// ParentURL returns the URL for the parent object
func (stream Stream) ParentURL() string {
	if streamURL, err := url.Parse(stream.URL); err == nil {
		streamURL.Path = stream.ParentID.Hex()
		return streamURL.String()
	}

	return ""
}

// HasGrandparent returns TRUE if this Stream has a GrandparentID
func (stream Stream) HasGrandparent() bool {
	return len(stream.ParentIDs) > 1
}

// GrandParentID returns the ID of the parent of the parent of this Stream (if it exists)
func (stream Stream) GrandparentID() primitive.ObjectID {
	if parentIDsLength := len(stream.ParentIDs); parentIDsLength > 1 {
		return stream.ParentIDs[parentIDsLength-2]
	}

	return primitive.NilObjectID
}

// GrandparentURL returns the URL of the parent of the parent of this Stream (if it exists)
func (stream Stream) GrandparentURL() string {

	if grandparentID := stream.GrandparentID(); !grandparentID.IsZero() {
		if streamURL, err := url.Parse(stream.URL); err == nil {
			streamURL.Path = grandparentID.Hex()
			return streamURL.String()
		}
	}

	return ""
}

// SetAttributedTo sets the list of people that this Stream is attributed to
func (stream *Stream) SetAttributedTo(person PersonLink) {
	if stream.AttributedTo.IsEmpty() {
		stream.AttributedTo = person
	}
}

// ActorLink returns a PersonLink object that represents this Stream as an ActivityPub "actor"
func (stream Stream) ActorLink() PersonLink {

	return PersonLink{
		Name:       stream.Label,
		ProfileURL: stream.URL,
		IconURL:    stream.IconURL,
	}
}

/******************************************
 * Webhook Interface
 ******************************************/

// GetWebhookData returns the data for this
// Stream that will be sent to a webhook
func (stream Stream) GetWebhookData() mapof.Any {
	return mapof.Any{
		"streamId":     stream.StreamID.Hex(),
		"url":          stream.URL,
		"template":     stream.TemplateID,
		"iconUrl":      stream.IconURL,
		"label":        stream.Label,
		"attributedTo": stream.AttributedTo,
		"data":         stream.Data,
		"isFeatured":   stream.IsFeatured,
		"isPublished":  stream.IsPublished(),
		"publishDate":  stream.PublishDate,
		"createDate":   stream.CreateDate,
		"updateDate":   stream.UpdateDate,
		"deleteDate":   stream.DeleteDate,
	}
}

// CopyFrom sets all values in this Stream to match the values in the provided Stream
func (stream *Stream) CopyFrom(other Stream) {
	stream.StreamID = other.StreamID
	stream.ParentID = other.ParentID
	stream.ParentIDs = other.ParentIDs
	stream.Rank = other.Rank
	stream.RankAlt = other.RankAlt
	stream.StartDate = other.StartDate
	stream.EndDate = other.EndDate
	stream.NavigationID = other.NavigationID
	stream.TemplateID = other.TemplateID
	stream.ParentTemplateID = other.ParentTemplateID
	stream.StateID = other.StateID
	stream.SocialRole = other.SocialRole
	stream.Groups = other.Groups
	stream.Circles = other.Circles
	stream.Products = other.Products
	stream.PrivilegeIDs = other.PrivilegeIDs
	stream.DefaultAllow = other.DefaultAllow
	stream.URL = other.URL
	stream.Token = other.Token
	stream.Label = other.Label
	stream.Summary = other.Summary
	stream.IconURL = other.IconURL
	stream.Icon = other.Icon
	stream.Context = other.Context
	stream.InReplyTo = other.InReplyTo
	stream.AttributedTo = other.AttributedTo
	stream.Content = other.Content
	stream.Widgets = other.Widgets
	stream.Hashtags = other.Hashtags
	stream.Data = other.Data
	stream.Syndication = other.Syndication
	stream.PublishDate = other.PublishDate
	stream.UnPublishDate = other.UnPublishDate
	stream.IsFeatured = other.IsFeatured
	stream.Journal = other.Journal
}
