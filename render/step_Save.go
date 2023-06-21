package render

import (
	"io"
	"text/template"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
)

// StepSave represents an action-step that can save changes to any object
type StepSave struct {
	Comment *template.Template
}

func (step StepSave) Get(renderer Renderer, _ io.Writer) PipelineBehavior {
	return nil
}

// Post saves the object to the database
func (step StepSave) Post(renderer Renderer, _ io.Writer) PipelineBehavior {

	modelService := renderer.service()
	object := renderer.object()
	comment := executeTemplate(step.Comment, renderer)

	if setter, ok := modelService.(service.AuthorSetter); ok {
		if err := setter.SetAuthor(object, renderer.AuthenticatedID()); err != nil {
			return Halt().WithError(derp.Wrap(err, "render.StepSave.Post", "Error setting author"))
		}
	}

	// Try to update the stream
	if err := modelService.ObjectSave(object, comment); err != nil {
		return Halt().WithError(derp.Wrap(err, "render.StepSave.Post", "Error saving model object"))
	}

	return nil
}
