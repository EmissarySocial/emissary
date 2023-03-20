package model

import (
	"mime"
	"strconv"
	"strings"

	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/list"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Attachment represents a file that has been uploaded to the software
type Attachment struct {
	AttachmentID primitive.ObjectID `bson:"_id"`        // ID of this Attachment
	ObjectID     primitive.ObjectID `bson:"objectId"`   // ID of the Stream that owns this Attachment
	ObjectType   string             `bson:"objectType"` // Type of object that owns this Attachment
	Original     string             `bson:"original"`   // Original filename uploaded by user
	Rank         int                `bson:"rank"`       // The sort order to display the attachments in.
	Height       int                `bson:"height"`     // Image height (if applicable)
	Width        int                `bson:"width"`      // Image width (if applicable)

	journal.Journal `bson:"journal"` // Journal entry for fetch compatability
}

// NewAttachment returns a fully initialized Attachment object.
func NewAttachment(objectType string, objectID primitive.ObjectID) Attachment {
	return Attachment{
		AttachmentID: primitive.NewObjectID(),
		ObjectType:   objectType,
		ObjectID:     objectID,
	}
}

/******************************************
 * data.Object Interface
 ******************************************/

// ID returns the primary key of this object
func (attachment *Attachment) ID() string {
	return attachment.AttachmentID.Hex()
}

/******************************************
 * Other Methods
 ******************************************/

func (attachment *Attachment) URL() string {
	switch attachment.ObjectType {
	case AttachmentTypeStream:
		return "/" + attachment.ObjectID.Hex() + "/attachments/" + attachment.AttachmentID.Hex()

	case AttachmentTypeUser:
		return "/@" + attachment.ObjectID.Hex() + "/pub/avatar/" + attachment.AttachmentID.Hex()
	}

	return ""
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
	return list.Slash(attachment.MimeType()).First()
}

func (attachment *Attachment) AspectRatio() string {

	if attachment.Width == 0 {
		return ""
	}

	return strconv.Itoa(attachment.Height / attachment.Width)
}
