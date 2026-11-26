package build

import (
	"io"

	"github.com/benpate/derp"
)

// StepDeleteArchive is a Step that deletes a Stream's ZIP archive from the export cache
type StepDeleteArchive struct {
	Token string
}

// Get does nothing
func (step StepDeleteArchive) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	return Continue()
}

// Post deletes the archive file
func (step StepDeleteArchive) Post(builder Builder, _ io.Writer) PipelineBehavior {

	streamBuilder, isStreamBuilder := builder.(Stream)

	if !isStreamBuilder {
		return Halt().WithError(derp.BadRequest("build.StepGetArchive.Get", "The `export` step can only be called on a `Stream` builder"))
	}

	streamArchiveService := streamBuilder.factory().StreamArchive()
	streamID := streamBuilder._stream.StreamID

	if err := streamArchiveService.Delete(streamID, step.Token); err != nil {
		return Halt().WithError(err)
	}

	return Continue()
}
