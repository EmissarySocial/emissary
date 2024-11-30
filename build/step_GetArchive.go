package build

import (
	"io"
	"net/http"
	"strings"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
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

	const location = "build.StepGetArchive.Get"

	streamBuilder, isStreamBuilder := builder.(*Stream)

	if !isStreamBuilder {
		return Halt().WithError(derp.NewBadRequestError("build.StepGetArchive.Get", "The `export` step can only be called on a `Stream` builder"))
	}

	streamArchiveService := streamBuilder.factory().StreamArchive()
	streamID := streamBuilder._stream.StreamID

	log.Trace().Str("location", location).Msg("Locating archive in cache...")

	// If the export file already exists, then return it
	if streamArchiveService.Exists(streamID, step.Token) {
		log.Trace().Str("location", location).Msg("Cache reports that stream archive exists ...")

		if err := streamArchiveService.Read(streamID, step.Token, writer); err == nil {
			filename := strings.ReplaceAll(streamBuilder._stream.Label, `"`, "") + ".zip"

			log.Trace().Str("location", location).Msg("Delivered archive from cache.")
			return Halt().
				AsFullPage().
				WithContentType("application/x-zip").
				WithHeader("Content-Disposition", `attachment; filename="`+filename+`"`)
		} else {
			log.Error().Str("location", location).Err(err).Msg("Error reading archive from cache.")
			derp.Report(err)
		}
	}

	log.Trace().Str("location", location).Msg("Archive does not exist.  Fall through to create a new archive....")

	// If we don't already have a file, try to create one using the task queue.
	q := streamBuilder.factory().Queue()
	task := queue.NewTask("MakeStreamArchive", mapof.Any{
		"host":        streamBuilder.Hostname(),
		"streamId":    streamBuilder.StreamID(),
		"token":       step.Token,
		"depth":       step.Depth,
		"json":        step.JSON,
		"attachments": step.Attachments,
		"metadata":    step.Metadata,
	})

	if err := q.Publish(task); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error publishing task", task))
	}

	log.Trace().Str("location", location).Msg("Archive task added to the to queue.")

	// Return a "please wait" message and tell the user to come back later.
	return step.FileNotReady(builder, writer)
}

// Post removes the object from the database (likely using a soft-delete, though)
func (step StepGetArchive) Post(builder Builder, _ io.Writer) PipelineBehavior {
	return Continue()
}

func (step StepGetArchive) FileNotReady(builder Builder, writer io.Writer) PipelineBehavior {
	_, _ = writer.Write([]byte(`<div>Export file is being generated. Please <a href="javascript:window.location.reload()">try again</a> in one minute.</div>`))

	log.Trace().Str("location", "build.StepGetArchive.FileNotReady").Msg("Sending 'Retry-After' message to browser.")
	return Halt().
		AsFullPage().
		WithStatusCode(http.StatusServiceUnavailable).
		WithHeader("Retry-After", "60").
		WithContentType("text/html")
}
