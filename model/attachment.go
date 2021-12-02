package model

import (
	"mime"

	"github.com/benpate/data/journal"
	"github.com/benpate/list"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Attachment represents a file that has been uploaded to the software
type Attachment struct {
	AttachmentID primitive.ObjectID `json:"attachmentId" bson:"_id"`      // ID of this Attachment
	StreamID     primitive.ObjectID `json:"streamId"     bson:"streamId"` // ID of the Stream that owns this Attachment
	Filename     string             `json:"filename"     bson:"filename"` // Name of the file that is currently stored on the filesystem
	Original     string             `json:"original"     bson:"original"` // Original filename uploaded by user

	journal.Journal `json:"journal" bson:"journal"` // Journal entry for fetch compatability
}

func NewAttachment() Attachment {
	return Attachment{
		AttachmentID: primitive.NewObjectID(),
	}
}

// ID returns the primary key of this object
func (attachment *Attachment) ID() string {
	return attachment.AttachmentID.Hex()
}

// Extension returns the file extension of the attached file
func (attachment *Attachment) Extension() string {
	return list.LastDelim(attachment.Filename, ".")
}

// OriginalExtension returns the file extension of the original filename
func (attachment *Attachment) OriginalExtension() string {
	return list.LastDelim(attachment.Original, ".")
}

// MimeType returns the mime-type of the attached file
func (attachment *Attachment) MimeType() string {
	return mime.TypeByExtension(attachment.Extension())
}
