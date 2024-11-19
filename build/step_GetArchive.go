package build

import (
	"io"
	"net/http"
	"strings"

	"github.com/benpate/derp"
	"github.com/davecgh/go-spew/spew"
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

	// If the export file already exists, then return it
	if streamArchiveService.Exists(streamID, step.Token) {

		spew.Dump("StepGetArchve.  writer is null?", writer == nil)
		if err := streamArchiveService.Read(streamID, step.Token, writer); err != nil {
			return Halt().WithError(err)
		}

		filename := strings.ReplaceAll(streamBuilder._stream.Label, `"`, "") + ".zip"
		return Halt().AsFullPage().WithContentType("application/x-zip").WithHeader("Content-Disposition", `attachment; filename="`+filename+`"`)
	}

	// If we don't already have a file, try to create one, then wait for it to be created.
	stepMakeArchive := StepMakeArchive(step)
	stepMakeArchive.Post(builder, writer)

	// Return a "please wait" message and tell the user to come back later.
	return step.FileNotReady(builder, writer)
}

// Post removes the object from the database (likely using a soft-delete, though)
func (step StepGetArchive) Post(builder Builder, _ io.Writer) PipelineBehavior {
	return Continue()
}

func (step StepGetArchive) FileNotReady(builder Builder, writer io.Writer) PipelineBehavior {
	_, _ = writer.Write([]byte(`<div>Export file is being generated. Please <a href="javascript:window.location.reload()">try again</a> in one minute.</div>`))

	return Halt().
		AsFullPage().
		WithStatusCode(http.StatusServiceUnavailable).
		WithHeader("Refresh", "60").
		WithContentType("text/html")
}
