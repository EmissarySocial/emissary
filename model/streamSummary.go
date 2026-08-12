package model

import (
	"time"

	"github.com/EmissarySocial/emissary/tools/datetime"
	"github.com/benpate/geo"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StreamSummary represents a partial stream record (used for lists)
type StreamSummary struct {
	ObjectID       primitive.ObjectID `bson:"_id"`                    // Unique identifier of this Stream.  (NOT USED PUBLICLY)
	ParentObjectID primitive.ObjectID `bson:"parentId"`               // Unique identifier of the "parent" stream. (NOT USED PUBLICLY)
	Token          string             `bson:"token"`                  // Unique value that identifies this element in the URL
	TemplateID     string             `bson:"templateId"`             // Unique identifier (name) of the Template to use when building this Stream in HTML.
	URL            string             `bson:"url,omitempty"`          // URL of the original document
	Label          string             `bson:"label,omitempty"`        // Label/Title of the document
	Summary        string             `bson:"summary,omitempty"`      // Brief summary of the document
	Content        Content            `bson:"content,omitempty"`      // Content of the document
	Data           mapof.Any          `bson:"data,omitempty"`         // Additional data that is specific to the Template used to build this Stream
	Icon           string             `bson:"icon,omitempty"`         // Icon name for this document
	IconURL        string             `bson:"iconUrl,omitempty"`      // URL of the icon image for this document
	AttributedTo   PersonLink         `bson:"attributedTo,omitempty"` // List of people who are attributed to this document
	Hashtags       sliceof.String     `bson:"hashtags,omitempty"`     // DEPRECATED: superseded by Tags. See projects/TAGS-UNIFICATION.md
	Tags           TagList            `bson:"tags,omitempty"`         // All tags associated with this document, with their AS2 types
	InReplyTo      string             `bson:"inReplyTo,omitempty"`    // If this stream is a reply to another stream or web page, then this links to the original document.
	StartDate      datetime.DateTime  `bson:"startDate,omitempty"`    // Date when this stream was published
	PublishDate    int64              `bson:"publishDate"`            // Unix epoch SECONDS when this stream was published (mirrors Stream.PublishDate)
	UnPublishDate  int64              `bson:"unpublishDate"`          // Unix epoch SECONDS when this stream should be removed from public view (mirrors Stream.UnPublishDate)
	Rank           int                `bson:"rank"`                   // If Template uses a custom sort order, then this is the value used to determine the position of this Stream.
	Shuffle        int64              `bson:"shuffle"`                // Random value used to shuffle the order of Streams in a list
	Location       geo.Address        `bson:"location"`               // Physical location associated with this document
	IsFeatured     bool               `bson:"isFeatured"`             // If this Stream is "featured" then it will be displayed in a special location on the page.
	CreateDate     int64              `bson:"createDate"`             // Unix epoch MILLISECONDS when this stream was created (journal field; mirrors Stream.CreateDate)
}

// NewStreamSummary returns a fully initialized StreamSummary object.
func NewStreamSummary() StreamSummary {

	streamID := primitive.NewObjectID()

	return StreamSummary{
		ObjectID:       streamID,
		Token:          streamID.Hex(),
		ParentObjectID: primitive.NilObjectID,
	}
}

// StreamSummaryFields returns the database columns that must be loaded to populate a StreamSummary
func StreamSummaryFields() []string {
	return []string{"_id", "parentId", "token", "templateId", "url", "label", "summary", "content", "data", "icon", "iconUrl", "hashtags", "tags", "attributedTo", "inReplyTo", "publishDate", "unpublishDate", "rank", "shuffle", "isFeatured", "startDate", "createDate", "location"}
}

// Fields returns the database columns that must be loaded to populate a StreamSummary
// It is part of the FieldLister interface
func (summary StreamSummary) Fields() []string {
	return StreamSummaryFields()
}

/*************************************
 * Other Data Accessors
 *************************************/

// ID returns the unique identifier for this Stream (in string format)
func (summary StreamSummary) ID() string {
	return summary.ObjectID.Hex()
}

// Name returns the label of this Stream
func (summary StreamSummary) Name() string {
	return summary.Label
}

// Description returns the summary of this Stream
func (summary StreamSummary) Description() string {
	return summary.Summary
}

// Author returns the PersonLink of whoever this Stream is attributed to
func (summary StreamSummary) Author() PersonLink {
	return summary.AttributedTo
}

// StreamID returns the unique identifier for this Stream (in string format)
func (summary StreamSummary) StreamID() string {
	return summary.ObjectID.Hex()
}

// ParentID returns the unique identifier of this Stream's parent (in string format)
func (summary StreamSummary) ParentID() string {
	return summary.ParentObjectID.Hex()
}

// ContentHTML returns this Stream's content, rendered as HTML
func (summary StreamSummary) ContentHTML() string {
	return summary.Content.HTML
}

// ContentRaw returns this Stream's content in its original, unrendered format
func (summary StreamSummary) ContentRaw() string {
	return summary.Content.Raw
}

// IsPublished returns TRUE if this Stream is inside its publication window right now
func (summary StreamSummary) IsPublished() bool {
	now := time.Now().Unix()
	return (summary.PublishDate < now) && (summary.UnPublishDate > now)

}
