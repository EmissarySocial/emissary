package render

import (
	"io"

	"github.com/benpate/datatype"
)

// StepRefreshPage represents an action-step that forwards the user to a new page.
type StepRefreshPage struct {
	BaseStep
}

// NewStepRefreshPage returns a fully initialized StepRefreshPage object
func NewStepRefreshPage(stepInfo datatype.Map) (StepRefreshPage, error) {
	return StepRefreshPage{}, nil
}

// Post updates the stream with approved data from the request body.
func (step StepRefreshPage) Post(_ Factory, renderer Renderer, _ io.Writer) error {
	renderer.context().Response().Header().Set("HX-Trigger", `{"closeModal":"", "refreshPage":""}`)
	return nil
}
