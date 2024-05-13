package build

import (
	"io"

	"github.com/benpate/derp"
)

// StepProcessTags is an action step that adds tags to a stream, either by scanning the content, or by
// calculating template values
type StepProcessTags struct {
	Paths []string
}

// Get builds the HTML for this step - either a modal template selector, or the embedded edit form
func (step StepProcessTags) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return nil
}

func (step StepProcessTags) Post(builder Builder, buffer io.Writer) PipelineBehavior {

	// Get a Stream Builder
	streamBuilder, ok := builder.(*Stream)

	if !ok {
		return Halt().WithError(derp.NewInternalError("builder.StepProcessTags.Post", "This step can only be used in a stream builder"))
	}

	// Calculate Tags
	stream := streamBuilder._stream
	streamService := builder.factory().Stream()
	streamService.CalcTagsFromPaths(stream, step.Paths...)
	return Continue()
}
