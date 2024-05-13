package build

import (
	"io"

	"github.com/benpate/derp"
)

// StepServerRedirect represents an action-step that continues building the output stream as
// a GET request to a new action.
type StepServerRedirect struct {
	On     string // "get" or "post" or "both"
	Action string
}

func (step StepServerRedirect) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	if step.On == "post" {
		return nil
	}

	return step.redirect(builder, buffer)
}

// Post updates the stream with approved data from the request body.
func (step StepServerRedirect) Post(builder Builder, _ io.Writer) PipelineBehavior {
	if step.On == "get" {
		return nil
	}

	return step.redirect(builder, builder.response())
}

// redirect creates a new builder on this object with the requested Action and then continues as a GET request.
func (step StepServerRedirect) redirect(builder Builder, buffer io.Writer) PipelineBehavior {

	newBuilder, err := builder.clone(step.Action)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepServerRedirect.Redirect", "Error creating new builder"))
	}

	result, err := newBuilder.Render()

	if err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepServerRedirect.Redirect", "Error building new page"))
	}

	if _, err := buffer.Write([]byte(result)); err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepServerRedirect.Redirect", "Error writing output buffer"))
	}

	return nil
}
