package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StreamSummary represents a partial stream record (used for lists)
type StreamSummary struct {
	StreamID    primitive.ObjectID `path:"streamId"       json:"streamId"            bson:"_id"`                 // Unique identifier of this Stream.  (NOT USED PUBLICLY)
	ParentID    primitive.ObjectID `path:"parentId"       json:"parentId"            bson:"parentId"`            // Unique identifier of the "parent" stream. (NOT USED PUBLICLY)
	Token       string             `path:"token"          json:"token"               bson:"token"`               // Unique value that identifies this element in the URL
	TemplateID  string             `path:"templateId"     json:"templateId"          bson:"templateId"`          // Unique identifier (name) of the Template to use when rendering this Stream in HTML.
	Document    DocumentLink       `path:"document"       json:"document"            bson:"document"`            // Link to the object that this stream is about
	InReplyTo   DocumentLink       `path:"inReplyTo"      json:"inReplyTo,omitempty" bson:"inReplyTo,omitempty"` // If this stream is a reply to another stream or web page, then this links to the original document.
	PublishDate int64              `path:"publishDate"    json:"publishDate"         bson:"publishDate"`         // Date when this stream was published
	Rank        int                `path:"rank"           json:"rank"                bson:"rank"`                // If Template uses a custom sort order, then this is the value used to determine the position of this Stream.
}

// TODO: MEDIUM: Lots of cleanup needed here.  InReplyTo should be migrated -> ReplyTo.
// TODO: MEDIUM: Is Origin even needed anymore, now that the Activity object will live in inboxes/outboxes?
// TODO: MEDIUM: Built-in fields should be migrated to a DocumentLink structure.

// NewStream returns a fully initialized Stream object.
func NewStreamSummary() StreamSummary {

	streamID := primitive.NewObjectID()

	return StreamSummary{
		StreamID: streamID,
		Token:    streamID.Hex(),
		ParentID: primitive.NilObjectID,
	}
}

func StreamSummaryFields() []string {
	return []string{"_id", "parentId", "token", "templateId", "document", "inReplyTo", "publishDate", "rank"}
}

func (streamSummary StreamSummary) Fields() []string {
	return StreamSummaryFields()
}
