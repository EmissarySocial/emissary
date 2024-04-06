package model

import (
	"mime"
	"strconv"
	"strings"

	"github.com/benpate/data/journal"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/list"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Attachment represents a file that has been uploaded to the software
type Attachment struct {
	AttachmentID primitive.ObjectID `bson:"_id"`         // ID of this Attachment
	ObjectID     primitive.ObjectID `bson:"objectId"`    // ID of the Stream that owns this Attachment
	ObjectType   string             `bson:"objectType"`  // Type of object that owns this Attachment
	Original     string             `bson:"original"`    // Original filename uploaded by user
	MediaType    string             `bson:"mediaType"`   // MIME type of the file
	Category     string             `bson:"category"`    // Category of the file (defined by the Template)
	Label        string             `bson:"label"`       // User-defined label for the attachment
	Description  string             `bson:"description"` // User-defined description for the attachment
	URL          string             `bson:"url"`         // URL where the file is stored
	Status       string             `bson:"status"`      // Status of the attachment (READY, WORKING)
	Height       int                `bson:"height"`      // Height of the media file (if applicable)
	Width        int                `bson:"width"`       // Width of the media file (if applicable)
	Duration     int                `bbson:"duration"`   // Duration of the media file (if applicable)
	Rank         int                `bson:"rank"`        // The sort order to display the attachments in.

	journal.Journal `json:"-" bson:",inline"` // Journal entry for fetch compatability
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

func (attachment *Attachment) CalcURL(host string) string {

	if attachment.ObjectType == AttachmentObjectTypeUser {
		return host + "/@" + attachment.ObjectID.Hex() + "/pub/avatar/" + attachment.AttachmentID.Hex()
	}

	return host + "/" + attachment.ObjectID.Hex() + "/attachments/" + attachment.AttachmentID.Hex()
}

func (attachment *Attachment) SetURL(host string) {
	attachment.URL = attachment.CalcURL(host)
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

func (attachment Attachment) HasDimensions() bool {
	if attachment.Width == 0 {
		return false
	}

	if attachment.Height == 0 {
		return false
	}

	return true
}

func (attachment Attachment) JSONLD() map[string]any {

	result := map[string]any{
		vocab.PropertyType:      vocab.ObjectTypeImage, // TODO: Expand this to videos, audios, etc.
		vocab.PropertyMediaType: attachment.DownloadMimeType(),
		vocab.PropertyURL:       attachment.URL,
	}

	if attachment.HasDimensions() {
		result["width"] = attachment.Width
		result["height"] = attachment.Height
	}

	// TODO: Alt Text
	// TODO: Blurhash

	return result
}
