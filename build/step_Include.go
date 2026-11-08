package build

import (
	"io"

	"github.com/benpate/derp"
)

// StepInclude is a Step that runs another action's steps on the current builder
type StepInclude struct {
	Action string
	Method string
}

// Get runs the included action during a GET request, unless its method is "post". Implements the Step interface.
func (step StepInclude) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	if step.Method != "post" {
		return step.execute(builder, buffer, ActionMethodGet)
	}

	return nil
}

// Post runs the included action during a POST request, unless its method is "get". Implements the Step interface.
func (step StepInclude) Post(builder Builder, buffer io.Writer) PipelineBehavior {

	if step.Method != "get" {
		return step.execute(builder, buffer, ActionMethodPost)
	}

	return nil
}

// execute runs the included action's steps on this builder, using the request's own method
func (step StepInclude) execute(builder Builder, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "build.StepInclude.execute"

	action, ok := builder.actions()[step.Action]

	if !ok {
		return Halt().WithError(derp.BadRequest(location, "Action not found", step.Action))
	}

	result := Pipeline(action.Steps).Execute(builder.factory(), builder, buffer, actionMethod)
	return UseResult(result)
}
