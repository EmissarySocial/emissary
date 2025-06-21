package build

import (
	"io"
	"text/template"

	"github.com/benpate/derp"
)

// StepSave is a Step that can save changes to any object
type StepSave struct {
	Comment *template.Template
	Method  string
}

func (step StepSave) Get(builder Builder, _ io.Writer) PipelineBehavior {

	if (step.Method == "get") || (step.Method == "both") {
		return step.do(builder)
	}

	return Continue()
}

// Post saves the object to the database
func (step StepSave) Post(builder Builder, _ io.Writer) PipelineBehavior {

	if (step.Method == "post") || (step.Method == "both") {
		return step.do(builder)
	}

	return Continue()
}

// Post saves the object to the database
func (step StepSave) do(builder Builder) PipelineBehavior {

	const location = "build.StepSave.Post"

	modelService := builder.service()
	object := builder.object()
	comment := executeTemplate(step.Comment, builder)

	// Try to update the stream
	if err := modelService.ObjectSave(object, comment); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error saving model object"))
	}

	return Continue()
}
