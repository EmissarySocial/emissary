package build

import (
	"io"

	"github.com/benpate/derp"
)

// StepUnPublish is a Step that moves a Stream into a state and saves it
type StepUnPublish struct {
	StateID string
}

// Get renders this step during a GET request. Implements the Step interface.
func (step StepUnPublish) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post moves the Stream into the step's state
func (step StepUnPublish) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepUnPublish.Post"

	// RULE: Require authentication to publish content
	if !builder.IsAuthenticated() {
		return Halt().WithError(derp.Unauthorized(location, "User must be authenticated to unpublish content"))
	}

	streamBuilder := builder.(Stream)
	factory := streamBuilder.factory()

	// Try to UnPublish the Stream from the search index
	searchResultService := factory.SearchResult()

	if err := searchResultService.DeleteByURL(builder.session(), streamBuilder._stream.URL); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Deleting search result", streamBuilder._stream.URL))
	}

	// Move the Stream into its new state.  The save queues the task that tells followers.
	streamService := factory.Stream()

	if err := streamService.UnPublish(builder.session(), streamBuilder._stream, step.StateID); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unpublishing stream", streamBuilder._stream.StreamID))
	}

	return nil
}
