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

	condition, err := step.evaluateCondition(builder)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error evaluating condition"))
	}

	if condition {
		result := Pipeline(step.Then).Execute(factory, builder, buffer, method)
		result.Error = derp.Wrap(result.Error, location, "Error executing 'then' sub-steps")
		return UseResult(result)
	}

	result := Pipeline(step.Otherwise).Execute(factory, builder, buffer, method)
	result.Error = derp.Wrap(result.Error, location, "Error executing 'otherwise' sub-steps")
	return UseResult(result)
}

// evaluateCondition executes the conditional template and
func (step StepIfCondition) evaluateCondition(builder Builder) (bool, error) {

	var buffer bytes.Buffer

	if err := step.Condition.Execute(&buffer, builder); err != nil {
		return false, derp.Wrap(err, "build.execute", "Error executing template")
	}

	return convert.Bool(buffer.String()), nil
}
