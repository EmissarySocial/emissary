package build

import (
	"io"
	"net/http"
	"strings"

	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/rs/zerolog/log"
)

// StepGetArchive is a Step that can delete a Stream from the Domain
type StepGetArchive struct {
	Token       string
	Depth       int
	JSON        bool
	Attachments bool
	Metadata    [][]map[string]any
}

// Get displays a customizable confirmation form for the delete
func (step StepGetArchive) Get(builder Builder, writer io.Writer) PipelineBehavior {

	const location = "build.StepGetArchive.Get"

	streamBuilder, isStreamBuilder := builder.(Stream)

	if !isStreamBuilder {
		return Halt().WithError(derp.BadRequest("build.StepGetArchive.Get", "The `export` step can only be called on a `Stream` builder"))
	}

	streamArchiveService := streamBuilder.factory().StreamArchive()
	streamID := streamBuilder._stream.StreamID

	log.Trace().Str("location", location).Msg("Locating archive in cache...")

	if exists, ready := streamArchiveService.Exists(streamID, step.Token); !ready {

		if !exists {

			log.Trace().Str("location", location).Msg("Archive does not exist.  Creating now.")

			// If we don't already have a file, try to create one using the task queue.
			// (This is a GET request — no transaction — so postcommit publishes immediately.)
			postcommit.Publish(builder.session(), streamBuilder.factory().Queue(), "MakeStreamArchive", mapof.Any{
				"hostname":    streamBuilder.Hostname(),
				"streamId":    streamBuilder.StreamID(),
				"token":       step.Token,
				"depth":       step.Depth,
				"json":        step.JSON,
				"attachments": step.Attachments,
				"metadata":    step.Metadata,
			})
		}

		log.Trace().Str("location", location).Msg("Archive is not ready.  Please wait.")
		return step.FileNotReady(builder, writer)
	}

	// If the export file already exists and is ready to use, then return it
	log.Trace().Str("location", location).Msg("Stream archive exists. Sending response to client...")

	if err := streamArchiveService.Read(streamID, step.Token, writer); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Reading archive from cache"))
	}

	// Add HTTP headers to the response.
	filename := strings.ReplaceAll(streamBuilder._stream.Label, `"`, "") + ".zip"

	return Halt().
		AsFullPage().
		WithContentType("application/x-zip").
		WithHeader("Content-Disposition", `attachment; filename="`+filename+`"`)
}

// Post removes the object from the database (likely using a soft-delete, though)
func (step StepGetArchive) Post(builder Builder, _ io.Writer) PipelineBehavior {
	return Continue()
}

// FileNotReady tells the caller that the archive is still being generated, and to retry shortly
func (step StepGetArchive) FileNotReady(builder Builder, writer io.Writer) PipelineBehavior {
	_, _ = writer.Write([]byte(`<div>Export file is being generated. Please <a href="javascript:window.location.reload()">try again</a> in one minute.</div>`))

	return Halt().
		AsFullPage().
		WithStatusCode(http.StatusAccepted).
		WithHeader("Retry-After", "60").
		WithContentType("text/html")
}
