package render

import (
	"io"

	"github.com/benpate/derp"
)

// StepSave represents an action-step that can save changes to any object
type StepSave struct {
	Comment string
}

func (step StepSave) Get(renderer Renderer, _ io.Writer) error {
	return nil
}

func (step StepSave) UseGlobalWrapper() bool {
	return true
}

// Post saves the object to the database
func (step StepSave) Post(renderer Renderer) error {

	object := renderer.object()

	// Try to update the stream
	if err := renderer.service().ObjectSave(object, step.Comment); err != nil {
		return derp.Wrap(err, "render.StepSave.Post", "Error saving model object")
	}

	return nil
}
