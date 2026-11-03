package build

import (
	"bytes"
	"io"
	"text/template"

	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
)

// StepIfCondition is a Step that can update the data.DataMap custom data stored in a Stream
type StepIfCondition struct {
	Condition *template.Template
	Then      []step.Step
	Otherwise []step.Step
}

// Get displays a form where users can update stream data
func (step StepIfCondition) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodGet)
}

// Post updates the stream with approved data from the request body.
func (step StepIfCondition) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodPost)
}

// Get displays a form where users can update stream data
func (step StepIfCondition) execute(builder Builder, buffer io.Writer, method ActionMethod) PipelineBehavior {

	const location = "builder.StepIfCondition.execute"

	factory := builder.factory()

	condition, err := evaluateCondition(step.Condition, builder)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Evaluating condition"))
	}

	if condition {
		result := Pipeline(step.Then).Execute(factory, builder, buffer, method)
		result.Error = derp.WrapIF(result.Error, location, "Executing 'then' sub-steps")

		return UseResult(result)
	}

	result := Pipeline(step.Otherwise).Execute(factory, builder, buffer, method)
	result.Error = derp.WrapIF(result.Error, location, "Executing 'otherwise' sub-steps")
	return UseResult(result)
}

// evaluateCondition renders a condition template, and returns TRUE only when it renders exactly "true".
// The "if" and "require" steps share it, so that they always agree on what is true.
func evaluateCondition(condition *template.Template, builder Builder) (bool, error) {

	var buffer bytes.Buffer

	if err := condition.Execute(&buffer, builder); err != nil {
		return false, derp.Wrap(err, "build.execute", "Executing template")
	}

	return convert.Bool(buffer.String()), nil
}
