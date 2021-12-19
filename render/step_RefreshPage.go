package render

import (
	"io"

	"github.com/benpate/datatype"
)

// StepRefreshPage represents an action-step that forwards the user to a new page.
type StepRefreshPage struct{}

// NewStepRefreshPage returns a fully initialized StepRefreshPage object
func NewStepRefreshPage(stepInfo datatype.Map) StepRefreshPage {
	return StepRefreshPage{}
}

// Get displays a form where users can update stream data
func (step StepRefreshPage) Get(buffer io.Writer, renderer Renderer) error {
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepRefreshPage) Post(buffer io.Writer, renderer Renderer) error {
	renderer.context().Response().Header().Set("HX-Refresh", "true")
	return nil
}
