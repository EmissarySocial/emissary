package build

import (
	"archive/zip"
	"io"
	"strings"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/translate"
)

// StepExport represents an action-step that can delete a Stream from the Domain
type StepExport struct {
	Depth       int
	Attachments bool
	Metadata    []translate.Pipeline
}

// Get displays a customizable confirmation form for the delete
func (step StepExport) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepExport.Get"

	// Guarantee that we have a Stream builder
	streamBuilder, ok := builder.(*Stream)

	if !ok {
		err := derp.NewBadRequestError(location, "The `export` step can only be called on a `Stream` builder")
		return Halt().WithError(err)
	}

	// Create a ZIP file
	writer := zip.NewWriter(buffer)

	// Write the stream to the ZIP file
	streamService := streamBuilder.factory().Stream()
	if err := streamService.ExportZip(writer, nil, streamBuilder._stream, "", step.Depth, step.Attachments, step.Metadata); err != nil {
		return Halt().WithError(err)
	}

	// Send the ZIP file to the browser
	if err := writer.Close(); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error closing ZIP file"))
	}

	// Done.
	filename := strings.ReplaceAll(streamBuilder._stream.Label, `"`, "")
	filename = filename + ".zip"
	return Halt().AsFullPage().WithContentType("application/zip").WithHeader(`Content-Disposition`, `attachment; filename="`+filename+`"`)
}

// Post removes the object from the database (likely using a soft-delete, though)
func (step StepExport) Post(builder Builder, _ io.Writer) PipelineBehavior {
	return Continue()
}
