package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StreamSummary represents a partial stream record (used for lists)
type StreamSummary struct {
	ObjectID       primitive.ObjectID `json:"streamId"               bson:"_id"`                    // Unique identifier of this Stream.  (NOT USED PUBLICLY)
	ParentObjectID primitive.ObjectID `json:"parentId"               bson:"parentId"`               // Unique identifier of the "parent" stream. (NOT USED PUBLICLY)
	Token          string             `json:"token"                  bson:"token"`                  // Unique value that identifies this element in the URL
	TemplateID     string             `json:"templateId"             bson:"templateId"`             // Unique identifier (name) of the Template to use when rendering this Stream in HTML.
	URL            string             `json:"url,omitempty"          bson:"url,omitempty"`          // URL of the original document
	Label          string             `json:"label,omitempty"        bson:"label,omitempty"`        // Label/Title of the document
	Summary        string             `json:"summary,omitempty"      bson:"summary,omitempty"`      // Brief summary of the document
	Content        Content            `json:"content,omitempty"      bson:"content,omitempty"`      // Content of the document
	ImageURL       string             `json:"imageUrl,omitempty"     bson:"imageUrl,omitempty"`     // URL of the cover image for this document's image
	AttributedTo   PersonLink         `json:"attributedTo,omitempty" bson:"attributedTo,omitempty"` // List of people who are attributed to this document
	InReplyTo      string             `json:"inReplyTo,omitempty"    bson:"inReplyTo,omitempty"`    // If this stream is a reply to another stream or web page, then this links to the original document.
	PublishDate    int64              `json:"publishDate"            bson:"publishDate"`            // Date when this stream was published
	Rank           int                `json:"rank"                   bson:"rank"`                   // If Template uses a custom sort order, then this is the value used to determine the position of this Stream.
}

// TODO: MEDIUM: Lots of cleanup needed here.  InReplyTo should be migrated -> ReplyTo.
// TODO: MEDIUM: Is Origin even needed anymore, now that the Activity object will live in inboxes/outboxes?
// TODO: MEDIUM: Built-in fields should be migrated to a DocumentLink structure.

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
	return []string{"_id", "parentId", "token", "templateId", "url", "label", "summary", "content", "imageUrl", "attributedTo", "inReplyTo", "publishDate", "rank"}
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
