package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
)

// StepSave represents an action-step that can save changes to any object
type StepSave struct {
	comment string
}

// NewStepSave returns a fully initialized StepSave object
func NewStepSave(stepInfo datatype.Map) StepSave {

	return StepSave{
		comment: stepInfo.GetString("comment"),
	}
}

// Get does nothing.
func (step StepSave) Get(buffer io.Writer, renderer Renderer) error {
	return nil
}

// Post saves the object to the database
func (step StepSave) Post(buffer io.Writer, renderer Renderer) error {

	// Validate the object against the schema
	if err := renderer.schema().Validate(renderer.object()); err != nil {
		return derp.Wrap(err, "render.StepSave.Post", "Object has invalid data")
	}

	// Try to update the stream
	if err := renderer.service().ObjectSave(renderer.object(), step.comment); err != nil {
		return derp.Wrap(err, "render.StepSave.Post", "Error saving Stream")
	}

	return nil
}
