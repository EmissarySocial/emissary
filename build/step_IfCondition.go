package build

import (
	"bytes"
	"html/template"
	"io"
	"strings"

	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
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

	if step.evaluateCondition(builder) {
		result := Pipeline(step.Then).Execute(factory, builder, buffer, method)
		result.Error = derp.Wrap(result.Error, location, "Error executing 'then' sub-steps")
		return UseResult(result)
	}

	result := Pipeline(step.Otherwise).Get(factory, builder, buffer)
	result.Error = derp.Wrap(result.Error, location, "Error executing 'otherwise' sub-steps")
	return UseResult(result)
}

// evaluateCondition executes the conditional template and
func (step StepIfCondition) evaluateCondition(builder Builder) bool {

	var result bytes.Buffer

	if err := step.Condition.Execute(&result, builder); err != nil {
		return false
	}

	return (strings.TrimSpace(result.String()) == "true")
}
