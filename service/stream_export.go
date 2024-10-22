package service

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/counter"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/list"
)

func (service *Stream) ExportZip(writer *zip.Writer, stream *model.Stream, prefix string, depth int, withAttachments bool) error {

	const location = "service.Stream.ExportZip"

	// Determine the filename of the root item
	filename := list.ByDot(prefix)

	if (prefix == "") || (strings.HasSuffix(prefix, "/")) {
		// if this is the top file in a directory, then name it "info"
		// otherwise, we'll just add ".json" to the filename we've been given (below)
		filename = filename.PushTail("info")
	}

	streamData := service.JSONLD(stream)

	// EXPORT A JSON FILE
	{
		filenameJSON := filename.PushTail("json")

		// Create a file in the ZIP archive
		fileWriter, err := writer.Create(filenameJSON.String())

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
	if withAttachments {

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

			// Create a file in the ZIP archive
			fileHeader := zip.FileHeader{
				Name:   filename.String(),
				Method: zip.Store,
			}

			fileWriter, err := writer.CreateHeader(&fileHeader)

			if err != nil {
				return derp.Wrap(err, location, "Error creating attachment file")
			}

			// Write the file into the ZIP archive
			if err := service.mediaserver.Get(filespec, fileWriter); err != nil {
				return derp.Wrap(err, location, "Error getting attachment")
			}
		}
	}

	// Export children, if requested
	if depth > 0 {
		children, err := service.ListByParent(stream.StreamID)

		if err != nil {
			return derp.Wrap(err, location, "Error listing children")
		}

		index := 1
		child := model.NewStream()
		for children.Next(&child) {

			prefix := fmt.Sprintf("%02d.%s", index, child.Label)

			if depth > 1 {
				prefix = prefix + "/" // For deeper nesting, create a new directory
			}

			if err := service.ExportZip(writer, &child, prefix, depth-1, withAttachments); err != nil {
				return derp.Wrap(err, location, "Error exporting child")
			}

			index = index + 1
			child = model.NewStream()
		}
	}

	// Success??
	return nil
}
