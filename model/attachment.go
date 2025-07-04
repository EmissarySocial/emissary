package model

import (
	"mime"
	"net/url"
	"strconv"
	"strings"

	"github.com/benpate/data/journal"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/mediaserver"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/list"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Attachment represents a file that has been uploaded to the software
type Attachment struct {
	AttachmentID primitive.ObjectID `bson:"_id"`         // ID of this Attachment
	ObjectID     primitive.ObjectID `bson:"objectId"`    // ID of the object that owns this Attachment
	ObjectType   string             `bson:"objectType"`  // Type of object that owns this Attachment
	Original     string             `bson:"original"`    // Original filename uploaded by user
	Category     string             `bson:"category"`    // Category of the file (defined by the Template)
	Label        string             `bson:"label"`       // User-defined label for the attachment
	Description  string             `bson:"description"` // User-defined description for the attachment
	URL          string             `bson:"url"`         // URL where the file is stored
	Status       string             `bson:"status"`      // Status of the attachment (READY, WORKING)
	Rules        AttachmentRules    `bson:"rules"`       // Rules for downloading this attachment
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
		Rules:        NewAttachmentRules(),
	}
}

/******************************************
 * data.Object Interface
 ******************************************/

// ID returns the primary key of this object
func (attachment Attachment) ID() string {
	return attachment.AttachmentID.Hex()
}

/******************************************
 * AccessLister Interface
 ******************************************/

// State returns the current state of this Attachment.
// It is part of the AccessLister interface
func (attachment Attachment) State() string {
	return "default"
}

// IsAuthor returns TRUE if the provided UserID the author of this Attachment
// It is part of the AccessLister interface
func (attachment Attachment) IsAuthor(authorID primitive.ObjectID) bool {

	// if attachment.ObjectType == AttachmentObjectTypeStream {
	// TODO: What goes here??
	// }

	return false
}

// IsMyself returns TRUE if this object directly represents the provided UserID
// It is part of the AccessLister interface
func (attachment *Attachment) IsMyself(userID primitive.ObjectID) bool {

	if attachment.ObjectType == AttachmentObjectTypeUser {
		if attachment.ObjectID == userID {
			return true
		}
	}

	return false
}

// RolesToGroupIDs returns a slice of Group IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (attachment Attachment) RolesToGroupIDs(roleIDs ...string) Permissions {
	return NewPermissions()
}

// RolesToPrivilegeIDs returns a slice of Privileges (CircleIDs and ProductIDs) that
// grant access to any of the requested roles. It is part of the AccessLister interface
func (attachment Attachment) RolesToPrivilegeIDs(roleIDs ...string) Permissions {
	return NewPermissions()
}

/******************************************
 * Other Methods
 ******************************************/

func (attachment Attachment) CalcURL(host string) string {

	switch attachment.ObjectType {

	case AttachmentObjectTypeUser:
		return host + "/@" + attachment.ObjectID.Hex() + "/attachments/" + attachment.AttachmentID.Hex()

	case AttachmentObjectTypeDomain:
		return host + "/.domain/attachments/" + attachment.AttachmentID.Hex()

	default:
		return host + "/" + attachment.ObjectID.Hex() + "/attachments/" + attachment.AttachmentID.Hex()
	}
}

func (attachment Attachment) DownloadExtension() string {

	ext := strings.ToLower(attachment.OriginalExtension())

	switch ext {
	case ".jpg", ".jpeg", ".png":
		return ".webp"
	}

	return ext
}

func (attachment Attachment) DownloadMimeType() string {
	return mime.TypeByExtension(attachment.DownloadExtension())
}

// OriginalExtension returns the file extension of the original filename
func (attachment Attachment) OriginalExtension() string {
	return "." + list.Dot(attachment.Original).Last()
}

// MimeType returns the mime-type of the attached file
func (attachment Attachment) MimeType() string {
	return mime.TypeByExtension(attachment.OriginalExtension())
}

func (attachment Attachment) MimeCategory() string {
	return list.Slash(attachment.MimeType()).First()
}

func (attachment Attachment) AspectRatio() string {

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

func (attachment Attachment) FileSpec(address *url.URL) mediaserver.FileSpec {

	if address == nil {
		address = &url.URL{
			Path: "/" + attachment.AttachmentID.Hex(),
		}
	}

	return attachment.Rules.FileSpec(address, attachment.OriginalExtension())
}

func (attachment Attachment) JSONLD() map[string]any {

	result := map[string]any{
		vocab.PropertyType:      vocab.ObjectTypeDocument, // TODO: Expand this to videos, audios, etc?
		vocab.PropertyMediaType: attachment.DownloadMimeType(),
		vocab.PropertyURL:       attachment.URL,
		vocab.PropertyName:      first.String(attachment.Label, attachment.Description, attachment.Category),
	}

	if attachment.HasDimensions() {
		result["width"] = attachment.Width
		result["height"] = attachment.Height
	}

	// TODO: Blurhash
	// TODO: FocalPoint?? -> toot:focalPoint (http://joinmastodon.org/ns#focalPoint) https://docs.joinmastodon.org/spec/activitypub/
	// TODO: Icon (if available) -> icon: {type:"", mediaType:"", url:""}

	return result
}

/******************************************
 * Setter Methods
 ******************************************/

func (attachment *Attachment) SetRules(width int, height int, extensions []string) {
	attachment.Rules.Extensions = extensions
	attachment.Rules.Width = width
	attachment.Rules.Height = height
}
