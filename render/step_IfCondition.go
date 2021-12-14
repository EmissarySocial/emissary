package render

import (
	"io"
	"strings"

	"github.com/benpate/datatype"
)

// StepIfCondition represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepIfCondition struct {
	factory   Factory
	condition string
	then      []datatype.Map
	otherwise []datatype.Map
}

func NewStepIfCondition(factory Factory, stepInfo datatype.Map) StepIfCondition {

	return StepIfCondition{
		factory:   factory,
		condition: stepInfo.GetString("condition"),
		then:      stepInfo.GetSliceOfMap("then"),
		otherwise: stepInfo.GetSliceOfMap("else"),
	}
}

// Get displays a form where users can update stream data
func (step StepIfCondition) Get(buffer io.Writer, renderer Renderer) error {

	if step.evaluateCondition(renderer) {
		if len(step.then) > 0 {
			return DoPipeline(step.factory, renderer, buffer, step.then, ActionMethodGet)
		}

		return nil
	}

	if len(step.otherwise) > 0 {
		return DoPipeline(step.factory, renderer, buffer, step.otherwise, ActionMethodGet)
	}

	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepIfCondition) Post(buffer io.Writer, renderer Renderer) error {

	if step.evaluateCondition(renderer) {
		if len(step.then) > 0 {
			return DoPipeline(step.factory, renderer, buffer, step.then, ActionMethodPost)
		}
		return nil
	}

	if len(step.otherwise) > 0 {
		return DoPipeline(step.factory, renderer, buffer, step.otherwise, ActionMethodPost)
	}

	return nil
}

// evaluateCondition executes the conditional template and
func (step StepIfCondition) evaluateCondition(renderer Renderer) bool {
	result, _ := executeSingleTemplate(step.condition, renderer)
	return (strings.TrimSpace(result) == "true")
}
