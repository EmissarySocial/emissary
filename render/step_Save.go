package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/davecgh/go-spew/spew"
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

	// Try to update the stream
	if err := renderer.service().ObjectSave(renderer.object(), step.comment); err != nil {
		return derp.Wrap(err, "ghost.render.StepSave.Post", "Error saving Stream")
	}

	spew.Dump("=========== OBJECT SAVED ============")
	spew.Dump(renderer.service())

	return nil
}
