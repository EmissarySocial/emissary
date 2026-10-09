package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"maps"
	"net/url"
	"path"
	"slices"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service/content"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * StreamSource Attachments
 *
 * Files linked from a synchronized source are
 * copied in two stages.  The sync reconciles the
 * attachment records and rewrites the links,
 * inside its transaction.  A separate task then
 * downloads each file, outside any transaction.
 * See GIT-MARKDOWN-STREAM-ATTACHMENTS.md.
 ******************************************/

// importAttachments creates an attachment for every file the content links to, deletes the ones it
// no longer links to, and rewrites the content's links to the local copies.  It downloads nothing.
func (service *StreamSource) importAttachments(session data.Session, streamSource *model.StreamSource, body *model.Content) error {

	const location = "service.StreamSource.importAttachments"

	refs := content.FindAttachments(body.HTML, streamSource.URL)

	// RULE: Links past the cap stay as links to the remote file
	if len(refs) > content.MaxAttachments {
		refs = refs[:content.MaxAttachments]
	}

	existing, duplicates, err := service.importedAttachments(session, streamSource.StreamID)

	if err != nil {
		return derp.Wrap(err, location, "Loading imported attachments", streamSource.StreamID)
	}

	localURLs := make(map[string]string, len(refs))

	// Create or reuse an attachment for every link
	for rank, ref := range refs {

		attachment, exists := existing[ref.URL]
		delete(existing, ref.URL)

		if !exists {
			attachment = newImportedAttachment(streamSource.StreamID, ref.URL)
		}

		// A reused attachment is written only when its position in the document changed
		if !exists || (attachment.Rank != rank) {

			attachment.Rank = rank

			if err := service.attachmentService.Save(session, &attachment, "Linked from "+streamSource.URL); err != nil {
				return derp.Wrap(err, location, "Saving attachment", ref.URL)
			}
		}

		localURLs[ref.URL] = attachment.URL
	}

	// Remove every imported file that the content no longer links to
	for _, attachment := range append(duplicates, slices.Collect(maps.Values(existing))...) {
		if err := service.attachmentService.Delete(session, &attachment, "No longer linked from "+streamSource.URL); err != nil {
			return derp.Wrap(err, location, "Deleting attachment", attachment.AttachmentID)
		}
	}

	body.HTML = content.RewriteAttachments(body.HTML, streamSource.URL, localURLs)
	return nil
}

// importedAttachments returns a Stream's imported attachments keyed by the address each was copied
// from, plus any second attachment for an address already seen, which the caller deletes
func (service *StreamSource) importedAttachments(session data.Session, streamID primitive.ObjectID) (map[string]model.Attachment, []model.Attachment, error) {

	const location = "service.StreamSource.importedAttachments"

	attachments, err := service.attachmentService.QueryByCategory(session, model.AttachmentObjectTypeStream, streamID, model.AttachmentCategoryStreamSource)

	if err != nil {
		return nil, nil, derp.Wrap(err, location, "Querying attachments", streamID)
	}

	result := make(map[string]model.Attachment, len(attachments))
	duplicates := make([]model.Attachment, 0)

	for _, attachment := range attachments {

		if _, exists := result[attachment.SourceURL]; exists {
			duplicates = append(duplicates, attachment)
			continue
		}

		result[attachment.SourceURL] = attachment
	}

	return result, duplicates, nil
}

// queueUnfinishedAttachments queues a download for every imported attachment on a Stream that is
// not READY, including the ones a previous download could not finish
func (service *StreamSource) queueUnfinishedAttachments(session data.Session, streamID primitive.ObjectID) error {

	const location = "service.StreamSource.queueUnfinishedAttachments"

	attachments, err := service.attachmentService.QueryByCategory(session, model.AttachmentObjectTypeStream, streamID, model.AttachmentCategoryStreamSource)

	if err != nil {
		return derp.Wrap(err, location, "Querying attachments", streamID)
	}

	for _, attachment := range attachments {
		if attachment.Status != model.AttachmentStatusReady {
			service.PublishAttachmentTask(session, streamID, attachment.AttachmentID)
		}
	}

	return nil
}

// PublishAttachmentTask queues the download of one imported attachment's file
func (service *StreamSource) PublishAttachmentTask(session data.Session, streamID primitive.ObjectID, attachmentID primitive.ObjectID) {

	// RULE: Signed, because every sync re-queues the files that are not READY, and a burst of
	// webhook pings must collapse into one download per file
	postcommit.Publish(
		session,
		service.queue,
		TaskSyncStreamSourceAttachment,
		mapof.Any{
			"hostname":     service.hostname,
			"streamId":     streamID.Hex(),
			"attachmentId": attachmentID.Hex(),
		},
		queue.WithSignature("StreamSourceAttachment:"+attachmentID.Hex()),
	)
}

// ImportAttachment downloads the file behind one imported attachment and marks it READY.  It runs
// OUTSIDE a transaction, and writes nothing but that attachment.
func (service *StreamSource) ImportAttachment(ctx context.Context, session data.Session, streamID primitive.ObjectID, attachmentID primitive.ObjectID) error {

	const location = "service.StreamSource.ImportAttachment"

	// Load the attachment.  One that was deleted, or already finished, needs nothing.
	attachment := model.NewEmptyAttachment()

	if err := service.attachmentService.LoadByID(session, model.AttachmentObjectTypeStream, streamID, attachmentID, &attachment); err != nil {

		if derp.IsNotFound(err) {
			return nil
		}

		return derp.Wrap(err, location, "Loading attachment", attachmentID)
	}

	if attachment.Status == model.AttachmentStatusReady {
		return nil
	}

	// Find the adapter that reads this Stream's source.  A source that is gone has nothing to read.
	streamSource := model.NewStreamSource()

	if err := service.LoadByStreamID(session, streamID, &streamSource); err != nil {

		if derp.IsNotFound(err) {
			return nil
		}

		return derp.Wrap(err, location, "Loading StreamSource", streamID)
	}

	adapter, err := service.adapterFor(streamSource.Method)

	if err != nil {
		return derp.Wrap(err, location, "Unable to read this source", streamSource.StreamSourceID)
	}

	// Copy the file.  A failure the author has to fix is recorded on the attachment.
	if err := service.copyAttachment(ctx, adapter, &attachment); err != nil {
		return service.recordAttachmentFailure(session, &attachment, derp.Wrap(err, location, "Copying attachment", attachment.SourceURL))
	}

	attachment.Status = model.AttachmentStatusReady

	if err := service.saveAttachmentStatus(session, &attachment, "Downloaded"); err != nil {
		return derp.Wrap(err, location, "Saving attachment", attachmentID)
	}

	// Filed and delivered
	return nil
}

// copyAttachment downloads an attachment's file into the MediaServer, refusing a file whose bytes
// are not the kind its extension promised
func (service *StreamSource) copyAttachment(ctx context.Context, adapter content.Adapter, attachment *model.Attachment) error {

	const location = "service.StreamSource.copyAttachment"

	// RULE: The extension decides the kind, and the cap that comes with it
	kind := content.AttachmentKind(attachment.SourceURL)

	if kind == "" {
		return derp.BadRequest(location, "File type is not supported", attachment.Original)
	}

	reader, err := adapter.FetchFile(ctx, attachment.SourceURL, content.MaxAttachmentBytes(kind))

	if err != nil {
		return derp.Wrap(err, location, "Downloading file")
	}

	defer derp.ReportFunc(reader.Close)

	// Read the first bytes, which decide the file's real type
	header := make([]byte, 512)
	headerLength, err := io.ReadFull(reader, header)

	// A file shorter than the header is not an error; it is just a small file
	if (err != nil) && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return derp.Wrap(err, location, "Reading file")
	}

	header = header[:headerLength]
	contentType := model.DetectContentType(header, attachment.Original)

	// RULE: The bytes must agree with the extension.  An .mp4 that is not a video would otherwise
	// be served as one, inside a <video> player this server built.
	if !isAttachmentKind(kind, contentType) {
		return derp.BadRequest(location, "File contents do not match its extension", attachment.Original, contentType)
	}

	filename := attachment.AttachmentID.Hex()

	if err := service.mediaServer.Put(filename, io.MultiReader(bytes.NewReader(header), reader)); err != nil {

		// RULE: A failed write can leave part of a file behind, which would be served as the whole
		derp.Report(service.mediaServer.Delete(filename))
		return derp.Wrap(err, location, "Storing file")
	}

	attachment.ContentType = contentType
	return nil
}

// recordAttachmentFailure marks an attachment FAILED when the error is one the author has to fix,
// and returns the error either way.  Any other error leaves it WORKING, for the queue to retry.
func (service *StreamSource) recordAttachmentFailure(session data.Session, attachment *model.Attachment, err error) error {

	const location = "service.StreamSource.recordAttachmentFailure"

	// A 429 is a 4xx, but it is the origin asking for a pause, and the queue waits it out
	if isTooMany, _ := derp.IsTooManyRequests(err); isTooMany {
		return err
	}

	if !derp.IsClientError(err) {
		return err
	}

	attachment.Status = model.AttachmentStatusFailed

	if saveErr := service.saveAttachmentStatus(session, attachment, "Download failed"); saveErr != nil {
		derp.Report(derp.Wrap(saveErr, location, "Saving attachment", attachment.AttachmentID))
	}

	return err
}

// saveAttachmentStatus saves an attachment, and nudges every browser watching its Stream
func (service *StreamSource) saveAttachmentStatus(session data.Session, attachment *model.Attachment, note string) error {

	if err := service.attachmentService.Save(session, attachment, note); err != nil {
		return err
	}

	service.publishSSE(session, attachment.ObjectID)
	return nil
}

/******************************************
 * Helper Functions
 ******************************************/

// newImportedAttachment returns a WORKING attachment that stands in for a remote file until the
// file is downloaded
func newImportedAttachment(streamID primitive.ObjectID, sourceURL string) model.Attachment {

	result := model.NewAttachment(model.AttachmentObjectTypeStream, streamID)
	result.Category = model.AttachmentCategoryStreamSource
	result.SourceURL = sourceURL
	result.Original = attachmentFilename(sourceURL)
	result.Status = model.AttachmentStatusWorking

	return result
}

// attachmentFilename returns the filename at the end of an address, unescaped
func attachmentFilename(address string) string {

	parsed, err := url.Parse(address)

	if err != nil {
		return ""
	}

	return path.Base(parsed.Path)
}

// isAttachmentKind returns TRUE if a sniffed content type is the kind of file that an extension promised
func isAttachmentKind(kind string, contentType string) bool {

	switch kind {

	case model.AttachmentMediaTypeImage:
		// RULE: SVG can carry script, and would be served from this site's own origin
		return strings.HasPrefix(contentType, "image/") && (contentType != "image/svg+xml")

	case model.AttachmentMediaTypeVideo:
		return strings.HasPrefix(contentType, "video/")

	case model.AttachmentMediaTypeAudio:
		return strings.HasPrefix(contentType, "audio/")

	case model.AttachmentMediaTypeDocument:
		return contentType == "application/pdf"
	}

	return false
}
