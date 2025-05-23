package model

import (
	"time"

	"github.com/EmissarySocial/emissary/tools/datetime"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StreamSummary represents a partial stream record (used for lists)
type StreamSummary struct {
	ObjectID       primitive.ObjectID    `json:"streamId"               bson:"_id"`                    // Unique identifier of this Stream.  (NOT USED PUBLICLY)
	ParentObjectID primitive.ObjectID    `json:"parentId"               bson:"parentId"`               // Unique identifier of the "parent" stream. (NOT USED PUBLICLY)
	Token          string                `json:"token"                  bson:"token"`                  // Unique value that identifies this element in the URL
	TemplateID     string                `json:"templateId"             bson:"templateId"`             // Unique identifier (name) of the Template to use when building this Stream in HTML.
	URL            string                `json:"url,omitempty"          bson:"url,omitempty"`          // URL of the original document
	Label          string                `json:"label,omitempty"        bson:"label,omitempty"`        // Label/Title of the document
	Summary        string                `json:"summary,omitempty"      bson:"summary,omitempty"`      // Brief summary of the document
	Content        Content               `json:"content,omitempty"      bson:"content,omitempty"`      // Content of the document
	Data           mapof.Any             `json:"data,omitempty"         bson:"data,omitempty"`         // Additional data that is specific to the Template used to build this Stream
	Icon           string                `json:"icon,omitempty"         bson:"icon,omitempty"`         // Icon name for this document
	IconURL        string                `json:"iconUrl,omitempty"      bson:"iconUrl,omitempty"`      // URL of the icon image for this document
	AttributedTo   PersonLink            `json:"attributedTo,omitempty" bson:"attributedTo,omitempty"` // List of people who are attributed to this document
	InReplyTo      string                `json:"inReplyTo,omitempty"    bson:"inReplyTo,omitempty"`    // If this stream is a reply to another stream or web page, then this links to the original document.
	StartDate      datetime.DateTime     `json:"startDate,omitempty"    bson:"startDate,omitempty"`    // Date when this stream was published
	PublishDate    int64                 `json:"publishDate"            bson:"publishDate"`            // Date when this stream was published
	UnPublishDate  int64                 `json:"unpublishDate"          bson:"unpublishDate"`          // Date when this stream should be removed from public view
	Rank           int                   `json:"rank"                   bson:"rank"`                   // If Template uses a custom sort order, then this is the value used to determine the position of this Stream.
	Shuffle        int64                 `json:"shuffle"                bson:"shuffle"`                // Random value used to shuffle the order of Streams in a list
	Places         sliceof.Object[Place] `json:"places"                 bson:"places"`                 // List of places that are associated with this document
	IsFeatured     bool                  `json:"isFeatured"             bson:"isFeatured"`             // If this Stream is "featured" then it will be displayed in a special location on the page.
	CreateDate     int64                 `json:"createDate"             bson:"createDate"`             // Date when this stream was created
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
