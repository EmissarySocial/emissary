package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepWithStreamSource is a Step that switches to the StreamSource record attached to a Stream,
// and executes its sub-steps against it
type StepWithStreamSource struct {
	SubSteps []step.Step
}

// Get renders this step during a GET request. Implements the Step interface.
func (step StepWithStreamSource) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodGet)
}

// Post renders this step during a POST request. Implements the Step interface.
func (step StepWithStreamSource) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodPost)
}

// execute performs this step's work for either a GET or a POST
func (step StepWithStreamSource) execute(builder Builder, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "build.StepWithStreamSource.execute"

	// RULE: A StreamSource is found by the Stream it populates, so there is nothing to switch to
	// from any other builder.
	streamBuilder, isStreamBuilder := builder.(Stream)

	if !isStreamBuilder {
		err := derp.BadRequest(location, "The `with-stream-source` step can only be called on a `Stream` builder")
		return Halt().WithError(err)
	}

	template, exists := getTemplate(builder)

	if !exists {
		return Halt().WithError(derp.Internal(location, "This step cannot be used in this Renderer."))
	}

	// Collect required services and values
	factory := builder.factory()
	streamSource, err := step.getStreamSource(streamBuilder)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Getting StreamSource record"))
	}

	// Create a new builder tied to the StreamSource record
	subBuilder, err := NewModel(factory, builder.session(), builder.request(), builder.response(), template, &streamSource, builder.actionID())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Creating sub-builder"))
	}

	// Execute the build pipeline on the StreamSource record
	result := Pipeline(step.SubSteps).Execute(factory, subBuilder, buffer, actionMethod)
	result.Error = derp.WrapIF(result.Error, location, "Executing steps for child")

	return UseResult(result)
}

// getStreamSource loads the StreamSource attached to this Stream, or returns a new one
func (step StepWithStreamSource) getStreamSource(builder Stream) (model.StreamSource, error) {

	const location = "build.StepWithStreamSource.getStreamSource"

	streamSourceService := builder.factory().StreamSource()
	streamID := builder._stream.StreamID

	// RULE: The load target is a ZERO record, never NewStreamSource().  The constructor seeds
	// Config with a minted webhook token, and a decode MERGES into a map rather than replacing it
	// -- so a seeded key the stored document does not name would survive into a loaded record.
	var streamSource model.StreamSource

	if err := streamSourceService.LoadByStreamID(builder.session(), streamID, &streamSource); err != nil {

		// A Stream with no source yet is the CREATE case, not an error
		if derp.IsNotFound(err) {
			streamSource = model.NewStreamSource()

			// Save refuses a record that names no Stream, so a create without this would fail
			// validation instead of working
			streamSource.StreamID = streamID

			return streamSource, nil
		}

		return model.StreamSource{}, derp.Wrap(err, location, "Loading StreamSource", streamID)
	}

	return streamSource, nil
}
