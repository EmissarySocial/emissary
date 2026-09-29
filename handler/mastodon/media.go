package mastodon

import (
	"bytes"
	"image"
	_ "image/gif"  // registers GIF with image.DecodeConfig
	_ "image/jpeg" // registers JPEG with image.DecodeConfig
	_ "image/png"  // registers PNG with image.DecodeConfig
	"io"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// mediaCategory is the Attachment Category a Mastodon-API upload is filed under, so it can be
// told apart from attachments the web app's own upload steps create (icons, editorjs images).
const mediaCategory = "mastodon-media"

// https://docs.joinmastodon.org/methods/media/
//
// RULE: media uploads before the status exists, so this has no Stream yet -- it's filed
// under the caller's User until PostStatus re-parents it (see attachStatusMedia).
func PostMedia(serverFactory *server.Factory) func(model.Authorization, txn.PostMedia) (object.MediaAttachment, error) {

	const location = "handler.mastodon.PostMedia"

	return func(auth model.Authorization, t txn.PostMedia) (object.MediaAttachment, error) {

		if t.File == nil {
			return object.MediaAttachment{}, derp.BadRequest(location, "No file was uploaded")
		}

		factory, session, cancel, err := statusSession(serverFactory, t.Host, location)

		if err != nil {
			return object.MediaAttachment{}, err
		}

		defer cancel()

		source, err := t.File.Open()

		if err != nil {
			return object.MediaAttachment{}, derp.Wrap(err, location, "Opening uploaded file")
		}

		defer source.Close()

		reader, contentType, width, height, err := sniffMedia(source)

		if err != nil {
			return object.MediaAttachment{}, derp.Wrap(err, location, "Reading uploaded file")
		}

		attachment := model.NewAttachment(model.AttachmentObjectTypeUser, auth.UserID)
		attachment.Original = t.File.Filename
		attachment.ContentType = contentType
		attachment.Category = mediaCategory
		attachment.Description = t.Description
		attachment.Width = width
		attachment.Height = height

		if err := factory.MediaServer().Put(attachment.AttachmentID.Hex(), reader); err != nil {
			return object.MediaAttachment{}, derp.Wrap(err, location, "Saving uploaded file")
		}

		if err := factory.Attachment().Save(session, &attachment, "Uploaded via Mastodon API"); err != nil {
			return object.MediaAttachment{}, derp.Wrap(err, location, "Saving attachment record")
		}

		return attachmentToMediaAttachment(attachment), nil
	}
}

// sniffMedia detects a file's real content type and (for images) dimensions from its own bytes,
// never the client-supplied filename. The returned reader replays the full stream afterward.
func sniffMedia(source io.Reader) (io.Reader, string, int, int, error) {

	const location = "handler.mastodon.sniffMedia"

	header := make([]byte, 512)
	headerLength, err := io.ReadFull(source, header)

	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, "", 0, 0, derp.Wrap(err, location, "Reading file header")
	}

	header = header[:headerLength]
	contentType := http.DetectContentType(header)
	reader := io.MultiReader(bytes.NewReader(header), source)

	width, height := 0, 0

	// image.DecodeConfig only needs the header, already fully read above -- reassemble a
	// second reader from the same bytes rather than consuming the one the caller still needs.
	if config, _, err := image.DecodeConfig(bytes.NewReader(header)); err == nil {
		width = config.Width
		height = config.Height
	}

	return reader, contentType, width, height, nil
}

// https://docs.joinmastodon.org/methods/media/#get
//
// A client polls this while an upload is still processing; this server always finishes synchronously.
func GetMedia(serverFactory *server.Factory) func(model.Authorization, txn.GetMedia) (object.MediaAttachment, error) {

	const location = "handler.mastodon.GetMedia"

	return func(auth model.Authorization, t txn.GetMedia) (object.MediaAttachment, error) {

		factory, session, cancel, err := statusSession(serverFactory, t.Host, location)

		if err != nil {
			return object.MediaAttachment{}, err
		}

		defer cancel()

		attachment, err := loadCallerMedia(factory, session, auth, t.ID)

		if err != nil {
			return object.MediaAttachment{}, derp.Wrap(err, location, "Loading media", t.ID)
		}

		return attachmentToMediaAttachment(attachment), nil
	}
}

// https://docs.joinmastodon.org/methods/media/#update
//
// Lets the caller change a photo's alt text, whether or not it's attached to a post yet.
func PutMedia(serverFactory *server.Factory) func(model.Authorization, txn.PutMedia) (object.MediaAttachment, error) {

	const location = "handler.mastodon.PutMedia"

	return func(auth model.Authorization, t txn.PutMedia) (object.MediaAttachment, error) {

		factory, session, cancel, err := statusSession(serverFactory, t.Host, location)

		if err != nil {
			return object.MediaAttachment{}, err
		}

		defer cancel()

		attachment, err := loadCallerMedia(factory, session, auth, t.ID)

		if err != nil {
			return object.MediaAttachment{}, derp.Wrap(err, location, "Loading media", t.ID)
		}

		attachment.Description = t.Description

		if err := factory.Attachment().Save(session, &attachment, "Updated via Mastodon API"); err != nil {
			return object.MediaAttachment{}, derp.Wrap(err, location, "Saving attachment")
		}

		return attachmentToMediaAttachment(attachment), nil
	}
}

// loadCallerMedia loads a Mastodon-API media attachment the caller is allowed to see or edit:
// either their own still-pending upload (see PostMedia), or one already attached to a Stream
// they wrote (per userOwnsStream's own author-only rule -- editing alt text is a write).
func loadCallerMedia(factory *service.Factory, session data.Session, auth model.Authorization, mediaID string) (model.Attachment, error) {

	const location = "handler.mastodon.loadCallerMedia"

	attachmentID, err := primitive.ObjectIDFromHex(mediaID)

	if err != nil {
		return model.Attachment{}, derp.Wrap(err, location, "Invalid media ID", mediaID)
	}

	attachment := model.NewEmptyAttachment()

	if err := factory.Attachment().LoadByID(session, model.AttachmentObjectTypeUser, auth.UserID, attachmentID, &attachment); err == nil {
		if attachment.Category == mediaCategory {
			return attachment, nil
		}
	}

	if err := factory.Attachment().Load(session, exp.Equal("_id", attachmentID).AndEqual("objectType", model.AttachmentObjectTypeStream), &attachment); err != nil {
		return model.Attachment{}, derp.NotFound(location, "Media not found", mediaID)
	}

	stream := model.NewStream()

	if err := factory.Stream().LoadByID(session, attachment.ObjectID, &stream); err != nil {
		return model.Attachment{}, derp.NotFound(location, "Media not found", mediaID)
	}

	if err := userOwnsStream(&auth, &stream); err != nil {
		return model.Attachment{}, derp.Forbidden(location, "Media does not belong to this account", mediaID)
	}

	return attachment, nil
}

// attachStatusMedia re-parents the caller's pending media uploads onto the new Stream.
// RULE: refuses anything not already a pending upload owned by this caller.
func attachStatusMedia(factory *service.Factory, session data.Session, auth model.Authorization, stream *model.Stream, mediaIDs []string) error {

	const location = "handler.mastodon.attachStatusMedia"

	attachmentService := factory.Attachment()

	for _, mediaID := range mediaIDs {

		attachmentID, err := primitive.ObjectIDFromHex(mediaID)

		if err != nil {
			return derp.Wrap(err, location, "Invalid media ID", mediaID)
		}

		attachment := model.NewEmptyAttachment()

		if err := attachmentService.LoadByID(session, model.AttachmentObjectTypeUser, auth.UserID, attachmentID, &attachment); err != nil {
			return derp.Wrap(err, location, "Media not found, or already attached to another post", mediaID)
		}

		if attachment.Category != mediaCategory {
			return derp.BadRequest(location, "Media not found, or already attached to another post", mediaID)
		}

		attachment.ObjectType = model.AttachmentObjectTypeStream
		attachment.ObjectID = stream.StreamID

		if err := attachmentService.Save(session, &attachment, "Attached to status via Mastodon API"); err != nil {
			return derp.Wrap(err, location, "Attaching media to status", mediaID)
		}
	}

	return nil
}

// attachmentToMediaAttachment converts a saved Attachment into its Mastodon API equivalent.
func attachmentToMediaAttachment(attachment model.Attachment) object.MediaAttachment {

	mediaType := mediaAttachmentType(attachment.ContentType)

	previewURL := ""

	if mediaType == "image" {
		previewURL = attachment.URL
	}

	meta := map[string]any{}

	if attachment.Width > 0 && attachment.Height > 0 {
		meta["original"] = map[string]any{
			"width":  attachment.Width,
			"height": attachment.Height,
		}
	}

	return object.MediaAttachment{
		ID:          attachment.AttachmentID.Hex(),
		Type:        mediaType,
		URL:         attachment.URL,
		PreviewURL:  previewURL,
		Description: attachment.Description,
		Meta:        meta,
	}
}

// mediaAttachmentType derives the Mastodon media type [image|video|audio] from a real MIME type.
// Anything else falls back to "image" -- Mastodon defines no "unknown" category.
func mediaAttachmentType(contentType string) string {

	switch {

	case len(contentType) >= 6 && contentType[:6] == "video/":
		return "video"

	case len(contentType) >= 6 && contentType[:6] == "audio/":
		return "audio"

	default:
		return "image"
	}
}
