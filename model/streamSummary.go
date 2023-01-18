package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StreamSummary represents a partial stream record (used for lists)
type StreamSummary struct {
	ObjectID       primitive.ObjectID `json:"streamId"            bson:"_id"`                 // Unique identifier of this Stream.  (NOT USED PUBLICLY)
	ParentObjectID primitive.ObjectID `json:"parentId"            bson:"parentId"`            // Unique identifier of the "parent" stream. (NOT USED PUBLICLY)
	Token          string             `json:"token"               bson:"token"`               // Unique value that identifies this element in the URL
	TemplateID     string             `json:"templateId"          bson:"templateId"`          // Unique identifier (name) of the Template to use when rendering this Stream in HTML.
	Document       DocumentLink       `json:"document"            bson:"document"`            // Link to the object that this stream is about
	InReplyTo      DocumentLink       `json:"inReplyTo,omitempty" bson:"inReplyTo,omitempty"` // If this stream is a reply to another stream or web page, then this links to the original document.
	PublishDate    int64              `json:"publishDate"         bson:"publishDate"`         // Date when this stream was published
	Rank           int                `json:"rank"                bson:"rank"`                // If Template uses a custom sort order, then this is the value used to determine the position of this Stream.
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
	return []string{"_id", "parentId", "token", "templateId", "document", "inReplyTo", "publishDate", "rank"}
}

func (summary StreamSummary) Fields() []string {
	return StreamSummaryFields()
}

/*************************************
 * Other Data Accessors
 *************************************/

func (summary StreamSummary) StreamID() string {
	return summary.ObjectID.Hex()
}

func (summary StreamSummary) ParentID() string {
	return summary.ParentObjectID.Hex()
}

func (summary StreamSummary) Label() string {
	return summary.Document.Label
}

func (summary StreamSummary) Summary() string {
	return summary.Document.Summary
}
