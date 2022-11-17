package model

import (
	"mime"
	"strings"

	"github.com/benpate/data/journal"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/list"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Attachment represents a file that has been uploaded to the software
type Attachment struct {
	AttachmentID primitive.ObjectID `                bson:"_id"`      // ID of this Attachment
	StreamID     primitive.ObjectID `                bson:"streamId"` // ID of the Stream that owns this Attachment
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
	}
}

/*******************************************
 * data.Object Interface
 *******************************************/

// ID returns the primary key of this object
func (attachment *Attachment) ID() string {
	return attachment.AttachmentID.Hex()
}

func (attachment *Attachment) GetObjectID(name string) (primitive.ObjectID, error) {

	switch name {
	case "attachmentId":
		return attachment.AttachmentID, nil
	case "streamId":
		return attachment.StreamID, nil
	}
	return primitive.NilObjectID, derp.NewInternalError("model.Attachment.GetObjectID", "Invalid property", name)
}

func (attachment *Attachment) GetString(name string) (string, error) {

	switch name {
	case "original":
		return attachment.Original, nil
	}
	return "", derp.NewInternalError("model.Attachment.GetString", "Invalid property", name)
}

func (attachment *Attachment) GetInt(name string) (int, error) {

	switch name {
	case "rank":
		return attachment.Rank, nil
	case "height":
		return attachment.Height, nil
	case "width":
		return attachment.Width, nil
	}

	return 0, derp.NewInternalError("model.Attachment.GetInt", "Invalid property", name)
}

func (attachment *Attachment) GetInt64(name string) (int64, error) {
	return 0, derp.NewInternalError("model.Attachment.GetInt64", "Invalid property", name)
}

func (attachment *Attachment) GetBool(name string) (bool, error) {
	return false, derp.NewInternalError("model.Attachment.GetBool", "Invalid property", name)
}

/*******************************************
 * OTHER FUNCTIONS
 *******************************************/

func (attachment *Attachment) URL() string {
	return "/" + attachment.StreamID.Hex() + "/attachments/" + attachment.AttachmentID.Hex()
}

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
	return "." + list.Dot(attachment.Original).Last()
}

// MimeType returns the mime-type of the attached file
func (attachment *Attachment) MimeType() string {
	return mime.TypeByExtension(attachment.OriginalExtension())
}

func (attachment *Attachment) MimeCategory() string {
	return list.Slash(attachment.MimeType()).Head()
}
