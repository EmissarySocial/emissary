package builder

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

func (step StepSave) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post saves the object to the database
func (step StepSave) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepSave.Post"

	modelService := builder.service()
	object := builder.object()
	comment := executeTemplate(step.Comment, builder)

	if setter, ok := modelService.(service.AuthorSetter); ok {
		if err := setter.SetAuthor(object, builder.AuthenticatedID()); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Error setting author"))
		}
	}

	// Try to update the stream
	if err := modelService.ObjectSave(object, comment); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error saving model object"))
	}

	return nil
}
