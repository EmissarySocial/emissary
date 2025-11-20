package build

import (
	"io"
	"text/template"

	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepSave is a Step that can save changes to any object
type StepSave struct {
	Comment *template.Template
	Method  string
	OnError []step.Step
}

func (step StepSave) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	if (step.Method == "get") || (step.Method == "both") {
		return step.do(builder, buffer, ActionMethodGet)
	}

	return Continue()
}

// Post saves the object to the database
func (step StepSave) Post(builder Builder, buffer io.Writer) PipelineBehavior {

	if (step.Method == "post") || (step.Method == "both") {
		return step.do(builder, buffer, ActionMethodPost)
	}

	return Continue()
}

// Post saves the object to the database
func (step StepSave) do(builder Builder, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "build.StepSave.Post"

	modelService := builder.service()
	object := builder.object()
	comment := executeTemplate(step.Comment, builder)

	// Try to update the stream
	err := modelService.ObjectSave(builder.session(), object, comment)

	// If success, then success
	if err == nil {
		return Continue()
	}

	// If there's no "on-error" pipeline, then fail in failure.
	if len(step.OnError) == 0 {
		return Halt().WithError(derp.Wrap(err, location, "Unable to save model object"))
	}

	// Otherwise, execute the "on-error" pipeline instead of failing.
	result := Pipeline(step.OnError).Execute(builder.factory(), builder, buffer, actionMethod)
	result.Error = derp.WrapIF(result.Error, location, "Error executing steps for child")

	return UseResult(result)
}
