package build

import (
	"io"

	"github.com/benpate/derp"
)

// StepStreamPromoteDraft is a Step that can copy the Container from a StreamDraft into its corresponding Stream
type StepStreamPromoteDraft struct {
	StateID string
}

func (step StepStreamPromoteDraft) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post copies relevant information from the draft into the primary stream, then deletes the draft
func (step StepStreamPromoteDraft) Post(builder Builder, _ io.Writer) PipelineBehavior {

	streamBuilder := builder.(Stream)

	factory := builder.factory()

	// Try to load the draft from the database, overwriting the stream already in the builder
	stream, err := factory.StreamDraft().Promote(builder.objectID(), step.StateID)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, "builder.StepStreamPromoteDraft.Post", "Error publishing draft"))
	}

	// Push the newly updated stream back to the builder so that subsequent
	// steps (e.g. publish) can use the correct data.
	streamBuilder._stream.CopyFrom(stream)

	return nil
}
