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

	const location = "build.StepProcessTags.Post"

	switch typed := builder.(type) {

	case Stream:
		stream := typed._stream
		streamService := builder.factory().Stream()
		streamService.CalculateTags(stream)
		return Continue()

	case Outbox:
		user := typed._user
		userService := builder.factory().User()
		userService.CalculateTags(user)
		return Continue()
	}

	return Halt().WithError(derp.NewInternalError(location, "This step can only be used in a Stream or User builder"))
}
