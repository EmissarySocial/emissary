package render

import (
	"bytes"
	"html/template"
	"io"
	"strings"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
)

// StepIfCondition represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepIfCondition struct {
	condition *template.Template
	then      Pipeline
	otherwise Pipeline

	BaseStep
}

func NewStepIfCondition(stepInfo datatype.Map) (StepIfCondition, error) {

	const location = "render.NewStepIfCondition"

	// Parse "condition" property
	condition, err := template.New("").Parse(stepInfo.GetString("condition"))

	if err != nil {
		return StepIfCondition{}, derp.Wrap(err, location, "Invalid 'condition'", stepInfo)
	}

	// Parse "then" property
	then, err := NewPipeline(stepInfo.GetSliceOfMap("then"))

	if err != nil {
		return StepIfCondition{}, derp.Wrap(err, location, "Invalid 'then'", stepInfo)
	}

	// Parse "else" property
	otherwise, err := NewPipeline(stepInfo.GetSliceOfMap("else"))

	if err != nil {
		return StepIfCondition{}, derp.Wrap(err, location, "Invalid 'else'", stepInfo)
	}

	return StepIfCondition{
		condition: condition,
		then:      then,
		otherwise: otherwise,
	}, nil
}

// Get displays a form where users can update stream data
func (step StepIfCondition) Get(factory Factory, renderer Renderer, buffer io.Writer) error {

	const location = "renderer.StepIfCondition.Get"

	if step.evaluateCondition(renderer) {
		if err := step.then.Get(factory, renderer, buffer); err != nil {
			return derp.Wrap(err, location, "Error executing 'then' sub-steps")
		}

		return nil
	}

	if err := step.otherwise.Get(factory, renderer, buffer); err != nil {
		return derp.Wrap(err, location, "Error executing 'otherwise' sub-steps")
	}

	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepIfCondition) Post(factory Factory, renderer Renderer, buffer io.Writer) error {

	const location = "renderer.StepIfCondition.Post"

	if step.evaluateCondition(renderer) {
		if err := step.then.Post(factory, renderer, buffer); err != nil {
			return derp.Wrap(err, location, "Error executing 'then' sub-steps")
		}

		return nil
	}

	if err := step.otherwise.Post(factory, renderer, buffer); err != nil {
		return derp.Wrap(err, location, "Error executing 'otherwise' sub-steps")
	}

	return nil
}

// evaluateCondition executes the conditional template and
func (step StepIfCondition) evaluateCondition(renderer Renderer) bool {

	var result bytes.Buffer

	if err := step.condition.Execute(&result, renderer); err != nil {
		return false
	}

	return (strings.TrimSpace(result.String()) == "true")
}
