package service

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/counter"
	"github.com/EmissarySocial/emissary/tools/ffmpeg"
	"github.com/benpate/derp"
	"github.com/benpate/mediaserver"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StreamArchive defines a service that manages all content streamArchives created and imported by Users.
type StreamArchive struct {
	streamService     *Stream
	attachmentService *Attachment
	mediaserver       mediaserver.MediaServer
	exportCache       afero.Fs
	host              string

	queue *queue.Queue
}

// NewStreamArchive returns a fully initialized StreamArchive service
func NewStreamArchive() StreamArchive {
	return StreamArchive{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *StreamArchive) Refresh(streamService *Stream, attachmentService *Attachment, mediaserver mediaserver.MediaServer, exportCache afero.Fs, queue *queue.Queue, host string) {
	service.streamService = streamService
	service.attachmentService = attachmentService
	service.mediaserver = mediaserver
	service.exportCache = exportCache
	service.queue = queue
	service.host = host
}

// Close stops any background processes controlled by this service
func (service *StreamArchive) Close() {
	// Nothin to do here.
}

/******************************************
 * CRUD Methods
 ******************************************/

// Exists returns TRUE if the specified ZIP archive exists in the export cache
func (service *StreamArchive) Exists(streamID primitive.ObjectID, token string) bool {
	filename := service.filename(streamID, token)
	exists, err := afero.Exists(service.exportCache, filename)
	return exists && (err == nil)
}

// ExistsTemp returns TRUE if a tempfile archive exists in the export cache
func (service *StreamArchive) ExistsTemp(streamID primitive.ObjectID, token string) bool {
	filename := service.filename(streamID, token) + ".tmp"
	exists, err := afero.Exists(service.exportCache, filename)
	return exists && (err == nil)
}

// Create makes a ZIP archive of a stream (and potentially its descendants)
// and saves it to the export cache
func (service *StreamArchive) Create(stream *model.Stream, options StreamArchiveOptions) error {

	const location = "service.StreamArchive.Create"

	filename := service.filename(stream.StreamID, options.Token)

	log.Trace().Str("location", location).Str("filename", filename).Msg("Creating ZIP archive...")

	// If the file already exists, then there is nothing to do.
	if service.Exists(stream.StreamID, options.Token) {
		log.Trace().Str("location", location).Str("filename", filename).Msg("ZIP archive already exists.")
		return nil
	}

	// If a temp file already exists, then there is nothing to do.
	if service.ExistsTemp(stream.StreamID, options.Token) {
		log.Trace().Str("location", location).Str("filename", filename).Msg("ZIP archive is being created.")
		return nil
	}

	// Create a temp file in the export cache
	if err := service.createTempfile(filename, stream, options); err != nil {
		return derp.Wrap(err, location, "Error creating ZIP archive")
	}

	// Move the temp file to the permanent location
	if err := service.exportCache.Rename(filename+".tmp", filename); err != nil {
		return derp.Wrap(err, location, "Error moving temp file into place")
	}

	// Great success.
	return nil
}

// createTempfile creates a ZIP archive of a stream in a temporary/working location.
func (service *StreamArchive) createTempfile(filename string, stream *model.Stream, options StreamArchiveOptions) (errorResult error) {

	const location = "service.StreamArchive.createTempfile"

	// Create a new file in the export cache
	file, err := service.exportCache.Create(filename + ".tmp")

	if err != nil {
		return derp.Wrap(err, location, "Error opening file", filename)
	}

	log.Trace().Str("location", location).Str("filename", filename).Msg("Opened file in export cache...")

	// Write the ZIP archive to the cached file
	zipWriter := zip.NewWriter(file)

	defer func() {
		zipWriter.Close()
		file.Close()

		if errorResult != nil {
			derp.Report(service.exportCache.Remove(filename + ".tmp"))
		}
	}()

	if err := service.writeToZip(zipWriter, nil, stream, "", options); err != nil {
		if err := service.Delete(stream.StreamID, options.Token); err != nil {
			errorResult = derp.Wrap(err, location, "Error writing/deleting ZIP archive")
			return
		}
		errorResult = derp.Wrap(err, location, "Error writing ZIP archive")
		return
	}

	log.Trace().Str("location", location).Str("filename", filename).Msg("Finished writing ZIP file to export cache.")
	errorResult = nil
	return
}

// Read retrieves a ZIP archive from the export cache.  If the file does not
// exist, then it returns an error
func (service *StreamArchive) Read(streamID primitive.ObjectID, token string, writer io.Writer) error {

	const location = "service.StreamArchive.Read"

	// Find the file in the export cache
	filename := service.filename(streamID, token)

	file, err := service.exportCache.Open(filename)

	if err != nil {
		return derp.Wrap(err, location, "Error opening file", filename)
	}

	defer file.Close()

	// Copy the fil to the destination
	if _, err = io.Copy(writer, file); err != nil {
		return derp.Wrap(err, location, "Error copying file", filename)
	}

	// Great success
	return nil
}

// Delete removes a ZIP archive from the export cache.
func (service *StreamArchive) Delete(streamID primitive.ObjectID, token string) error {

	const location = "service.StreamArchive.Delete"

	filename := service.filename(streamID, token)

	// If the file doesn't already exist, then there is nothing to do.
	if exists, _ := afero.Exists(service.exportCache, filename); !exists {
		return nil
	}

	// Remove the file from the exportCache
	if err := service.exportCache.Remove(filename); err != nil {
		return derp.Wrap(err, location, "Error deleting file", filename)
	}

	// Great success
	return nil
}

/******************************************
 * Helper Methods
 ******************************************/

// Some info on FFmpeg metadata
// https://gist.github.com/eyecatchup/0757b3d8b989fe433979db2ea7d95a01
// https://jmesb.com/how_to/create_id3_tags_using_ffmpeg
// https://wiki.multimedia.cx/index.php?title=FFmpeg_Metadata

// How to include album art...
// https://www.bannerbear.com/blog/how-to-add-a-cover-art-to-audio-files-using-ffmpeg/

// WriteZip exports a stream (and potentially its descendents) into a ZIP archive.
// The `depth` parameter indicates how many levels to traverse.
// The `pipelines` parameter provides rosetta.translate mappings for attachment metadata.
func (service *StreamArchive) writeToZip(zipWriter *zip.Writer, parent *model.Stream, stream *model.Stream, prefix string, options StreamArchiveOptions) error {

	const location = "service.StreamArchive.ExportZip"

	// Determine the filename of the root item
	filename := list.ByDot(prefix)

	if (prefix == "") || (strings.HasSuffix(prefix, "/")) {
		// if this is the top file in a directory, then name it "info"
		// otherwise, we'll just add ".json" to the filename we've been given (below)
		filename = filename.PushTail("info")
	}

	if options.JSON {
		streamData := service.streamService.JSONLD(stream)

		// EXPORT A JSON FILE
		filenameJSON := filename.PushTail("json")

		// Create a file in the ZIP archive
		fileWriter, err := zipWriter.Create(filenameJSON.String())

		if err != nil {
			return derp.Wrap(err, location, "Error creating JSON-LD file")
		}

		// Marshal the Stream data into JSON
		streamJSON, err := json.MarshalIndent(streamData, "", "\t")

		if err != nil {
			return derp.Wrap(err, location, "Error marshalling JSON-LD")
		}

		// Write the JSON-LD to the file
		if _, err := fileWriter.Write(streamJSON); err != nil {
			return derp.Wrap(err, location, "Error writing JSON-LD file")
		}
	}

	// Export attachments, if requested
	if options.Attachments {

		// Get all attachments for this Stream
		attachments, err := service.attachmentService.QueryByObjectID(model.AttachmentObjectTypeStream, stream.StreamID)

		if err != nil {
			return derp.Wrap(err, location, "Error listing attachments")
		}

		c := counter.NewCounter()

		// Count all attachments by category
		for _, attachment := range attachments {
			c.Add(attachment.Category)
		}

		// Add each attachment to the ZIP file
		for _, attachment := range attachments {

			// The filename is the prefix and the category
			filename := list.ByDot(prefix)

			if attachment.Category != "" {
				filename = filename.PushTail(attachment.Category)
			}

			// If there are multiple attachments in the same category, add the counter to the filename
			if count := c.Get(attachment.Category); count > 1 {
				filename = filename.PushTail(fmt.Sprintf("%02d", count))
			}

			if attachment.Label != "" {
				filename = filename.PushTail(attachment.Label)
			}

			// Add the corresponding extension to the filename
			filespec := attachment.FileSpec(nil)
			filespec.Bitrate = 128 // aligning bitrate with player to see if it helps cacheability

			filename = filename.PushTail(strings.TrimPrefix(filespec.Extension, "."))

			// Map attachment metadata
			metadata := mapof.NewString()

			if pipeline := options.Pipeline(); pipeline.NotEmpty() {

				inSchema := schema.New(schema.Object{
					Properties: schema.ElementMap{
						"parent": model.StreamSchema(),
						"stream": model.StreamSchema(),
					},
				})
				inObject := mapof.Any{
					"parent": parent,
					"stream": stream,
				}

				outSchema := schema.New(schema.Object{
					Wildcard: schema.String{},
				})

				if err := pipeline.Execute(inSchema, inObject, outSchema, &metadata); err != nil {
					return derp.Wrap(err, location, "Error processing metadata")
				}
			}

			// Make a pipe to transfer from MediaServer to the Metadata writer
			pipeReader, pipeWriter := io.Pipe()

			// Create a fileWriter in the ZIP archive
			fileHeader := zip.FileHeader{
				Name:   filename.String(),
				Method: zip.Store,
			}

			fileWriter, err := zipWriter.CreateHeader(&fileHeader)

			if err != nil {
				return derp.Wrap(err, location, "Error creating attachment file")
			}

			// Using separate goroutine to avoid deadlock between pipe reader/writer
			go func() {
				// Send the output from the MediaServer through FFmpeg one more time
				// to add metadata to the file *before* it's written to the ZIP archive
				if err := ffmpeg.SetMetadata(pipeReader, filespec.MimeType, metadata, fileWriter); err != nil {
					derp.Report(derp.Wrap(err, location, "Error setting metadata"))
				}
			}()

			// Write the file into the ZIP archive
			defer pipeWriter.Close()

			if err := service.mediaserver.Get(filespec, pipeWriter); err != nil {
				return derp.Wrap(err, location, "Error getting attachment")
			}
		}
	}

	// Export children, if requested
	if options.HasNext() {
		children, err := service.streamService.ListByParent(stream.StreamID)

		if err != nil {
			return derp.Wrap(err, location, "Error listing children")
		}

		index := 1
		child := model.NewStream()
		nextOptions := options.Next()
		for children.Next(&child) {

			prefix := fmt.Sprintf("%02d.%s", index, child.Label)

			if options.Depth > 1 {
				prefix = prefix + "/" // For deeper nesting, create a new directory
			}

			if err := service.writeToZip(zipWriter, stream, &child, prefix, nextOptions); err != nil {
				return derp.Wrap(err, location, "Error exporting child")
			}

			index = index + 1
			child = model.NewStream()
		}
	}

	// Success??
	return nil
}

func (service *StreamArchive) filename(streamID primitive.ObjectID, token string) string {
	return streamID.Hex() + "_" + token + ".zip"
}
