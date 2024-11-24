package service

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/counter"
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

var streamArchiveLock sync.Mutex

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

	// Look for the file in the export cache
	filename := service.filename(streamID, token)
	exists, _ := afero.Exists(service.exportCache, filename)
	return exists
}

// ExistsTemp returns TRUE if a tempfile archive exists in the export cache
func (service *StreamArchive) ExistsTemp(streamID primitive.ObjectID, token string) bool {

	// Look for the file in the export cache
	filename := service.filename(streamID, token) + ".tmp"
	exists, _ := afero.Exists(service.exportCache, filename)
	return exists
}

// Create makes a ZIP archive of a stream (and potentially its descendants)
// and saves it to the export cache
func (service *StreamArchive) Create(stream *model.Stream, options StreamArchiveOptions) error {

	const location = "service.StreamArchive.Create"

	filename := service.filename(stream.StreamID, options.Token)
	log.Trace().Str("location", location).Str("filename", filename).Msg("Started method. Waiting for lock...")

	// WriteLock for write operations - there can be only one.
	streamArchiveLock.Lock()
	defer streamArchiveLock.Unlock()

	// Remove orphaned files from the export cache
	derp.Report(service.exportCache.Remove(filename))

	log.Trace().Str("location", location).Str("filename", filename).Msg("Creating ZIP file in export cache...")

	// Create a new file in the export cache
	file, err := service.exportCache.Create(filename)

	if err != nil {
		return derp.Wrap(err, location, "Error opening file", filename)
	}

	defer file.Close()

	log.Trace().Str("location", location).Str("filename", filename).Msg("Writing to ZIP file in export cache...")

	// Write the ZIP archive to the cached file
	zipWriter := zip.NewWriter(file)

	defer zipWriter.Close()

	if err := service.writeToZip(zipWriter, nil, stream, "", options); err != nil {
		// if the write fails, then remove the file before exiting.
		derp.Report(service.exportCache.Remove(filename))
		return derp.Wrap(err, location, "Error writing ZIP archive")
	}

	log.Trace().Str("location", location).Str("filename", filename).Msg("ZIP file written to export cache successfully.")
	return nil
}

// Read retrieves a ZIP archive from the export cache.  If the file does not
// exist, then it returns an error
func (service *StreamArchive) Read(streamID primitive.ObjectID, token string, writer io.Writer) error {

	const location = "service.StreamArchive.Read"

	// Find the file in the export cache
	filename := service.filename(streamID, token)
	log.Trace().Str("location", location).Str("filename", filename).Msg("Reading ZIP archive...")

	file, err := service.exportCache.Open(filename)

	if err != nil {
		return derp.Wrap(err, location, "Error opening file", filename)
	}

	defer file.Close()

	// Copy the file to the destination
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
	log.Trace().Str("location", location).Str("filename", filename).Msg("Deleting ZIP archive...")

	// WriteLock for write operations - there can be only one.
	streamArchiveLock.Lock()
	defer streamArchiveLock.Unlock()

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
			filename = filename.PushTail(strings.TrimPrefix(filespec.Extension, "."))

			// Map attachment metadata
			if pipeline := options.Pipeline(); pipeline.NotEmpty() {

				// Don't use cache when we're adding custom metadata to files
				filespec.Cache = false

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

				if err := pipeline.Execute(inSchema, inObject, outSchema, &filespec.Metadata); err != nil {
					derp.Report(derp.Wrap(err, location, "Error processing metadata"))
					continue
				}
			}

			// Create a fileWriter in the ZIP archive
			fileHeader := zip.FileHeader{
				Name:   filename.String(),
				Method: zip.Store,
			}

			fileWriter, err := zipWriter.CreateHeader(&fileHeader)

			if err != nil {
				return derp.Wrap(err, location, "Error creating attachment file")
			}

			// Send the output from the MediaServer through FFmpeg one more time
			// to add metadata to the file *before* it's written to the ZIP archive
			if err := service.mediaserver.Process(filespec, fileWriter); err != nil {
				return derp.Wrap(err, location, "Error processing attachment", filespec)
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
