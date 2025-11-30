package model

import (
	"time"

	"github.com/EmissarySocial/emissary/tools/datetime"
	"github.com/benpate/geo"
	"github.com/benpate/rosetta/mapof"
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
	InReplyTo      string             `bson:"inReplyTo,omitempty"`    // If this stream is a reply to another stream or web page, then this links to the original document.
	StartDate      datetime.DateTime  `bson:"startDate,omitempty"`    // Date when this stream was published
	PublishDate    int64              `bson:"publishDate"`            // Date when this stream was published
	UnPublishDate  int64              `bson:"unpublishDate"`          // Date when this stream should be removed from public view
	Rank           int                `bson:"rank"`                   // If Template uses a custom sort order, then this is the value used to determine the position of this Stream.
	Shuffle        int64              `bson:"shuffle"`                // Random value used to shuffle the order of Streams in a list
	Location       geo.Address        `bson:"location"`               // Physical location associated with this document
	IsFeatured     bool               `bson:"isFeatured"`             // If this Stream is "featured" then it will be displayed in a special location on the page.
	CreateDate     int64              `bson:"createDate"`             // Date when this stream was created
}

// NewStream returns a fully initialized Stream object.
func NewStreamSummary() StreamSummary {

	streamID := primitive.NewObjectID()

	return StreamSummary{
		ObjectID:       streamID,
		Token:          streamID.Hex(),
		ParentObjectID: primitive.NilObjectID,
	}
}

func StreamSummaryFields() []string {
	return []string{"_id", "parentId", "token", "templateId", "url", "label", "summary", "content", "data", "icon", "iconUrl", "attributedTo", "inReplyTo", "publishDate", "unpublishDate", "rank", "shuffle", "isFeatured", "startDate", "createDate", "places"}
}

func (summary StreamSummary) Fields() []string {
	return StreamSummaryFields()
}

/*************************************
 * Other Data Accessors
 *************************************/

func (summary StreamSummary) ID() string {
	return summary.ObjectID.Hex()
}

func (summary StreamSummary) Name() string {
	return summary.Label
}

func (summary StreamSummary) Author() PersonLink {
	return summary.AttributedTo
}

func (summary StreamSummary) StreamID() string {
	return summary.ObjectID.Hex()
}

func (summary StreamSummary) ParentID() string {
	return summary.ParentObjectID.Hex()
}

func (summary StreamSummary) ContentHTML() string {
	return summary.Content.HTML
}

func (summary StreamSummary) ContentRaw() string {
	return summary.Content.Raw
}

func (summary StreamSummary) IsPublished() bool {
	now := time.Now().Unix()
	return (summary.PublishDate < now) && (summary.UnPublishDate > now)

}
