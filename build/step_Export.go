package build

import (
	"archive/zip"
	"io"
	"log"
	"os"

	"github.com/benpate/derp"
)

// StepExport represents an action-step that can delete a Stream from the Domain
type StepExport struct {
	Depth       int
	Attachments bool
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
	if err := streamService.ExportZip(writer, streamBuilder._stream, "", step.Depth, step.Attachments); err != nil {
		return Halt().WithError(err)
	}

	// Send the ZIP file to the browser
	if err := writer.Close(); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error closing ZIP file"))
	}

	// Done.
	filename := streamBuilder._stream.Label + ".zip"
	return Halt().AsFullPage().WithContentType("application/zip").WithHeader(`Content-Disposition`, `attachment; filename="`+filename+`"`)
}

// Post removes the object from the database (likely using a soft-delete, though)
func (step StepExport) Post(builder Builder, _ io.Writer) PipelineBehavior {
	return Continue()
}

func (step StepExport) test() {

	// Create a buffer to write our archive to.
	buf, err := os.OpenFile("./test-example.zip", os.O_WRONLY|os.O_CREATE, 0644)

	if err != nil {
		log.Fatal(err)
	}

	// Create a new zip archive.
	w := zip.NewWriter(buf)

	// Add some files to the archive.
	var files = []struct {
		Name, Body string
	}{
		{"readme.txt", "This archive contains some text files."},
		{"gopher.txt", "Gopher names:\nGeorge\nGeoffrey\nGonzo"},
		{"todo.txt", "Get animal handling licence.\nWrite more examples."},
	}
	for _, file := range files {
		f, err := w.Create(file.Name)
		if err != nil {
			log.Fatal(err)
		}
		_, err = f.Write([]byte(file.Body))
		if err != nil {
			log.Fatal(err)
		}
	}

	// Make sure to check the error on Close.
	if err := w.Close(); err != nil {
		log.Fatal(err)
	}
}
