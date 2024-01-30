package render

import (
	"io"

	"github.com/benpate/derp"
)

// StepDo represents an action-step that sends an HTMX 'forward' to a new page.
type StepDo struct {
	Action string
}

func (step StepDo) Get(renderer Renderer, buffer io.Writer) PipelineBehavior {

	const location = "render.StepDo.Get"

	action, ok := renderer.template().Actions[step.Action]

	if !ok {
		return Halt().WithError(derp.NewBadRequestError(location, "Action not found", step.Action))
	}

	result := Pipeline(action.Steps).Get(renderer.factory(), renderer, buffer)
	return UseResult(result)
}

// Post updates the stream with approved data from the request body.
func (step StepDo) Post(renderer Renderer, buffer io.Writer) PipelineBehavior {

	const location = "render.StepDo.Post"

	action, ok := renderer.template().Actions[step.Action]

	if !ok {
		return Halt().WithError(derp.NewBadRequestError(location, "Action not found", step.Action))
	}

	result := Pipeline(action.Steps).Post(renderer.factory(), renderer, buffer)
	return UseResult(result)
}
