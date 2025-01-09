package build

import (
	"io"

	"github.com/benpate/derp"
)

// StepDeleteArchive is a Step that can delete a Stream from the Domain
type StepDeleteArchive struct {
	Token string
}

// Get displays a customizable confirmation form for the delete
func (step StepDeleteArchive) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	return Continue()
}

// Post removes the object from the database (likely using a soft-delete, though)
func (step StepDeleteArchive) Post(builder Builder, _ io.Writer) PipelineBehavior {

	streamBuilder, isStreamBuilder := builder.(Stream)

	if !isStreamBuilder {
		return Halt().WithError(derp.NewBadRequestError("build.StepGetArchive.Get", "The `export` step can only be called on a `Stream` builder"))
	}

	streamArchiveService := streamBuilder.factory().StreamArchive()
	streamID := streamBuilder._stream.StreamID

	if err := streamArchiveService.Delete(streamID, step.Token); err != nil {
		return Halt().WithError(err)
	}

	return Continue()
}
