package build

import (
	"io"

	"github.com/benpate/derp"
)

// StepInclude is a Step that executes another action within this context.
type StepInclude struct {
	Action string
}

func (step StepInclude) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepInclude.Get"

	action, ok := builder.actions()[step.Action]

	if !ok {
		return Halt().WithError(derp.BadRequest(location, "Action not found", step.Action))
	}

	result := Pipeline(action.Steps).Get(builder.factory(), builder, buffer)
	return UseResult(result)
}

// Post updates the stream with approved data from the request body.
func (step StepInclude) Post(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepInclude.Post"

	action, ok := builder.actions()[step.Action]

	if !ok {
		return Halt().WithError(derp.BadRequest(location, "Action not found", step.Action))
	}

	result := Pipeline(action.Steps).Post(builder.factory(), builder, buffer)
	return UseResult(result)
}
