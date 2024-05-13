package build

import (
	"io"

	"github.com/benpate/derp"
)

// StepDo represents an action-step that sends an HTMX 'forward' to a new page.
type StepDo struct {
	Action string
}

func (step StepDo) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepDo.Get"

	action, ok := builder.actions()[step.Action]

	if !ok {
		return Halt().WithError(derp.NewBadRequestError(location, "Action not found", step.Action))
	}

	result := Pipeline(action.Steps).Get(builder.factory(), builder, buffer)
	return UseResult(result)
}

// Post updates the stream with approved data from the request body.
func (step StepDo) Post(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepDo.Post"

	action, ok := builder.actions()[step.Action]

	if !ok {
		return Halt().WithError(derp.NewBadRequestError(location, "Action not found", step.Action))
	}

	result := Pipeline(action.Steps).Post(builder.factory(), builder, buffer)
	return UseResult(result)
}
