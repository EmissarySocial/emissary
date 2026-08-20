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
	AttachmentID primitive.ObjectID `bson:"_id"`               // ID of this Attachment
	ObjectID     primitive.ObjectID `bson:"objectId"`          // ID of the object that owns this Attachment
	ObjectType   string             `bson:"objectType"`        // Type of object that owns this Attachment
	Original     string             `bson:"original"`          // Original filename uploaded by user
	ContentType  string             `bson:"contentType"`       // Media type sniffed from the file's own bytes.  Empty on records that predate content sniffing.
	Category     string             `bson:"category"`          // Category of the file (defined by the Template)
	Label        string             `bson:"label"`             // User-defined label for the attachment
	Description  string             `bson:"description"`       // User-defined description for the attachment
	URL          string             `bson:"url"`               // URL where the file is stored
	Status       string             `bson:"status"`            // Status of the attachment (READY, WORKING)
	Rules        AttachmentRules    `bson:"rules"`             // Rules for downloading this attachment
	Height       int                `bson:"height,omitzero"`   // Height of the media file (if applicable)
	Width        int                `bson:"width,omitzero"`    // Width of the media file (if applicable)
	Duration     int                `bson:"duration,omitzero"` // Duration of the media file (if applicable)
	Rank         int                `bson:"rank,omitzero"`     // The sort order to display the attachments in.

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

// NewEmptyAttachment returns a zero-value Attachment, with no ID and no rules
func NewEmptyAttachment() Attachment {
	return Attachment{}
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
	return false
}

// IsMyself returns TRUE if this object directly represents the provided UserID
// It is part of the AccessLister interface
func (attachment *Attachment) IsMyself(userID primitive.ObjectID) bool {

	if attachment.ObjectType == AttachmentObjectTypeUser {
		if !userID.IsZero() && attachment.ObjectID == userID {
			return true
		}
	}

	return false
}

// RolesToGroupIDs returns a slice of Group IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (attachment Attachment) RolesToGroupIDs(roleIDs ...string) Permissions {
	return defaultRolesToGroupIDs(primitive.NilObjectID, roleIDs...)
}

// RolesToPrivilegeIDs returns a slice of Privileges (CircleIDs and ProductIDs) that
// grant access to any of the requested roles. It is part of the AccessLister interface
func (attachment Attachment) RolesToPrivilegeIDs(roleIDs ...string) Permissions {
	return NewPermissions()
}

/******************************************
 * Other Methods
 ******************************************/

// CalcURL returns the public URL of this Attachment on the provided host
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

// DownloadExtension returns the file extension this Attachment is served with, which may differ from the original
func (attachment Attachment) DownloadExtension() string {

	ext := strings.ToLower(attachment.OriginalExtension())

	switch ext {
	case ".jpg", ".jpeg", ".png":
		return ".webp"
	}

	return ext
}

// DownloadMimeType returns the content type this Attachment is served with
func (attachment Attachment) DownloadMimeType() string {
	return mime.TypeByExtension(attachment.DownloadExtension())
}

// OriginalExtension returns the file extension of the original filename
func (attachment Attachment) OriginalExtension() string {
	return "." + list.Dot(attachment.Original).Last()
}

// OriginalMimeType returns the mime-type implied by the original filename.
func (attachment Attachment) OriginalMimeType() string {

	// The filename is chosen by whoever uploaded the file, so this type is a claim,
	// not a fact.  Use MimeType() unless you specifically need the claim.
	return mime.TypeByExtension(attachment.OriginalExtension())
}

// MimeType returns the mime-type of the attached file
func (attachment Attachment) MimeType() string {

	// Prefer the type sniffed from the file's own bytes...
	if attachment.ContentType != "" {
		return attachment.ContentType
	}

	// ...and fall back to the filename for Attachments that predate content sniffing.
	return attachment.OriginalMimeType()
}

// MimeCategory returns the first half of the mime-type of the attached file
func (attachment Attachment) MimeCategory() string {
	return list.Slash(attachment.MimeType()).First()
}

// CanServeInline returns TRUE if this Attachment's bytes may be rendered by a browser.
func (attachment Attachment) CanServeInline() bool {

	// FFmpeg decodes and re-generates every file it processes, so its output cannot carry
	// script regardless of what was uploaded.  Everything else is copied to the client
	// byte-for-byte, and could therefore be an HTML document wearing a costume.

	// RULE: MediaServer chooses the files it re-encodes from the original filename
	// (mediaserver.Process -> isFFmpegMediaType), so that same unverified name decides
	// this.  Otherwise we would describe bytes that MediaServer never generated.
	if !isInlineMediaCategory(list.Slash(attachment.OriginalMimeType()).First()) {
		return false
	}

	// Attachments that predate content sniffing have nothing left to check.
	if attachment.ContentType == "" {
		return true
	}

	// RULE: The sniffed contents must be a re-encodable category too.  MediaServer picks
	// the pipeline from the filename, so this is defense in depth: a file named ".png"
	// that actually contains HTML is downloaded rather than rendered, whatever FFmpeg
	// makes of it.  The two categories need not match each other -- FFmpeg generates
	// the output bytes either way.
	return isInlineMediaCategory(list.Slash(attachment.ContentType).First())
}

// isInlineMediaCategory returns TRUE if a mime category is one that MediaServer re-encodes.
func isInlineMediaCategory(mimeCategory string) bool {

	// This list mirrors mediaserver.isFFmpegMediaType.  If MediaServer learns to
	// re-encode a new category, this has to learn it too, or safe files start downloading.
	switch mimeCategory {

	case "image", "audio", "video":
		return true
	}

	return false
}

// AspectRatio returns the width-to-height ratio of this Attachment, or "auto" if its dimensions are unknown
func (attachment Attachment) AspectRatio() string {

	// RULE: Without both dimensions there is no ratio to compute
	if !attachment.HasDimensions() {
		return "auto"
	}

	// Templates drop this straight into a CSS `aspect-ratio`, so emit a real
	// number.  Integer division would round every ratio down to 1 or 0.
	ratio := float64(attachment.Width) / float64(attachment.Height)
	return strconv.FormatFloat(ratio, 'f', -1, 64)
}

// HasDimensions returns TRUE if this Attachment has both a width and a height
func (attachment Attachment) HasDimensions() bool {
	if attachment.Width == 0 {
		return false
	}

	if attachment.Height == 0 {
		return false
	}

	return true
}

// FileSpec returns the mediaserver FileSpec that describes how to render this Attachment for the provided URL
func (attachment Attachment) FileSpec(address *url.URL) mediaserver.FileSpec {

	if address == nil {
		address = &url.URL{
			Path: "/" + attachment.AttachmentID.Hex(),
		}
	}

	return attachment.Rules.FileSpec(address, attachment.OriginalExtension())
}

// JSONLD returns this Attachment as a JSON-LD map
func (attachment Attachment) JSONLD() map[string]any {

	result := map[string]any{
		vocab.PropertyType:      vocab.ObjectTypeDocument, // TODO: Expand this to videos, audios, etc?
		vocab.PropertyMediaType: attachment.DownloadMimeType(),
		vocab.PropertyURL:       attachment.URL,
		vocab.PropertyName:      first.String(attachment.Description, attachment.Label, attachment.Category),
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

// SetRules replaces the dimension and format rules that this Attachment is processed with
func (attachment *Attachment) SetRules(width int, height int, extensions []string) {
	attachment.Rules.Extensions = extensions
	attachment.Rules.Width = width
	attachment.Rules.Height = height
}
