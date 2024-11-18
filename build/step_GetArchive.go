package build

import (
	"io"
	"net/http"
	"time"

	"github.com/benpate/derp"
)

// StepGetArchive represents an action-step that can delete a Stream from the Domain
type StepGetArchive struct {
	Token       string
	Depth       int
	JSON        bool
	Attachments bool
	Metadata    [][]map[string]any
}

// Get displays a customizable confirmation form for the delete
func (step StepGetArchive) Get(builder Builder, writer io.Writer) PipelineBehavior {

	streamBuilder, isStreamBuilder := builder.(*Stream)

	if !isStreamBuilder {
		return Halt().WithError(derp.NewBadRequestError("build.StepGetArchive.Get", "The `export` step can only be called on a `Stream` builder"))
	}

	streamArchiveService := streamBuilder.factory().StreamArchive()
	streamID := streamBuilder._stream.StreamID

	for counter := range 6 {

		// If the export file already exists, then return it
		if streamArchiveService.Exists(streamID, step.Token) {

			if err := streamArchiveService.Read(streamID, step.Token, writer); err != nil {
				return Halt().WithError(err)
			}

			return Halt().AsFullPage().WithContentType("application/x-zip").WithHeader("Content-Disposition", "attachment; filename=\"archive.zip\"")
		}

		// First time through the loop, if we don't already have a file,
		// try to create one, then wait for it to be created.
		if counter == 0 {
			stepMakeArchive := StepMakeArchive(step)
			stepMakeArchive.Post(builder, writer)
		}

		// Wait up to (1 + 2 + 4 + 8 + 16) = 31 seconds for the file to generate before giving up.
		if counter < 5 {
			time.Sleep(time.Duration(2^counter) * time.Second)
		}
	}

	// Fall through to here means that the file is taking a LOONG time to generate.
	// So just return a "please wait" message and tell the user to come back later.
	return step.FileNotReady(builder, writer)
}

// Post removes the object from the database (likely using a soft-delete, though)
func (step StepGetArchive) Post(builder Builder, _ io.Writer) PipelineBehavior {
	return Continue()
}

func (step StepGetArchive) FileNotReady(builder Builder, writer io.Writer) PipelineBehavior {
	_, _ = writer.Write([]byte(`{"error": "Export file is still being generated. Please try again in a five minutes."}`))

	return Halt().
		AsFullPage().
		WithStatusCode(http.StatusServiceUnavailable).
		WithHeader("Retry-After", "300").
		WithContentType("application/json")
}
