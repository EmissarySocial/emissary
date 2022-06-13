package model

import (
	"mime"
	"strings"

	"github.com/benpate/data/journal"
	"github.com/benpate/list"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Attachment represents a file that has been uploaded to the software
type Attachment struct {
	AttachmentID primitive.ObjectID `                bson:"_id"`      // ID of this Attachment
	StreamID     primitive.ObjectID `                bson:"streamId"` // ID of the Stream that owns this Attachment
	Filename     string             `path:"filename" bson:"filename"` // Name of the file that is currently stored on the filesystem
	Original     string             `path:"original" bson:"original"` // Original filename uploaded by user
	Rank         int                `path:"rank"     bson:"rank"`     // The sort order to display the attachments in.
	Height       int                `path:"height"   bson:"height"`
	Width        int                `path:"width"    bson:"width"`

	journal.Journal `bson:"journal"` // Journal entry for fetch compatability
}

// NewAttachment returns a fully initialized Attachment object.
func NewAttachment(streamID primitive.ObjectID) Attachment {
	return Attachment{
		AttachmentID: primitive.NewObjectID(),
		StreamID:     streamID,
		Filename:     primitive.NewObjectID().Hex(),
	}
}

/*******************************************
 * DATA.OBJECT INTERFACE
 *******************************************/

// ID returns the primary key of this object
func (attachment *Attachment) ID() string {
	return attachment.AttachmentID.Hex()
}

/*******************************************
 * OTHER FUNCTIONS
 *******************************************/

func (attachment *Attachment) DownloadExtension() string {

	ext := strings.ToLower(attachment.OriginalExtension())

	switch ext {
	case ".jpg", ".jpeg", ".png":
		return ".webp"
	}

	return ext
}

func (attachment *Attachment) DownloadMimeType() string {
	return mime.TypeByExtension(attachment.DownloadExtension())
}

// OriginalExtension returns the file extension of the original filename
func (attachment *Attachment) OriginalExtension() string {
	return list.LastDelim(attachment.Original, ".")
}

// MimeType returns the mime-type of the attached file
func (attachment *Attachment) MimeType() string {
	return mime.TypeByExtension(attachment.OriginalExtension())
}

func (attachment *Attachment) MimeCategory() string {
	return list.Head(attachment.MimeType(), "/")
}
