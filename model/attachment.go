package model

import (
	"mime"
	"strings"

	"github.com/benpate/data/journal"
	"github.com/benpate/derp"
	"github.com/benpate/list"
	"github.com/benpate/path"
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

// GetPath implements the path.Getter interface, allowing named READ access to specific values
func (attachment *Attachment) GetPath(p path.Path) (interface{}, error) {
	return nil, derp.New(derp.CodeInternalError, "whisper.model.Attachment.GetPath", "unimplemented")
}

// GetPath implements the path.Getter interface, allowing named WRITE access to specific values
func (attachment *Attachment) SetPath(p path.Path, value interface{}) error {
	return derp.New(derp.CodeInternalError, "whisper.model.Attachment.GetPath", "unimplemented")
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
