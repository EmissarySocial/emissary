package model

import (
	"mime"
	"strings"

	"github.com/benpate/data/journal"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const AttachmentTypeStream = "Stream"

const AttachmentTypeUser = "User"

// Attachment represents a file that has been uploaded to the software
type Attachment struct {
	AttachmentID primitive.ObjectID `                bson:"_id"`        // ID of this Attachment
	ObjectID     primitive.ObjectID `                bson:"objectId"`   // ID of the Stream that owns this Attachment
	ObjectType   string             `                bson:"objectType"` // Type of object that owns this Attachment
	Original     string             `path:"original" bson:"original"`   // Original filename uploaded by user
	Rank         int                `path:"rank"     bson:"rank"`       // The sort order to display the attachments in.
	Height       int                `path:"height"   bson:"height"`
	Width        int                `path:"width"    bson:"width"`

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

func AttachmentSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"attachmentId": schema.String{Format: "objectId"},
			"objectId":     schema.String{Format: "objectId"},
			"objectType":   schema.String{Enum: []string{AttachmentTypeStream, AttachmentTypeUser}},
			"original":     schema.String{},
			"rank":         schema.Integer{},
			"height":       schema.Integer{},
			"width":        schema.Integer{},
		},
	}
}

/*******************************************
 * data.Object Interface
 *******************************************/

// ID returns the primary key of this object
func (attachment *Attachment) ID() string {
	return attachment.AttachmentID.Hex()
}

/*******************************************
 * Data Accessors
 *******************************************/

func (attachment *Attachment) GetObjectID(name string) (primitive.ObjectID, error) {

	switch name {
	case "attachmentId":
		return attachment.AttachmentID, nil
	case "objectId":
		return attachment.ObjectID, nil
	}
	return primitive.NilObjectID, derp.NewInternalError("model.Attachment.GetObjectID", "Invalid property", name)
}

func (attachment *Attachment) GetString(name string) (string, error) {

	switch name {
	case "objectType":
		return attachment.ObjectType, nil
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
	return list.Slash(attachment.MimeType()).Head()
}
