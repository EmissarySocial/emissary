package render

import (
	"io"
)

// StepReloadPage represents an action-step that forwards the user to a new page.
type StepReloadPage struct{}

func (step StepReloadPage) Get(renderer Renderer, _ io.Writer) error {
	return nil
}

func (step StepReloadPage) UseGlobalWrapper() bool {
	return true
}

// Post updates the stream with approved data from the request body.
func (step StepReloadPage) Post(renderer Renderer) error {
	renderer.context().Response().Header().Set("HX-Refresh", `true`)
	return nil
}
