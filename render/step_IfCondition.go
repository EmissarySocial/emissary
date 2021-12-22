package render

import (
	"io"
	"strings"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
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
			if err := DoPipeline(renderer, buffer, step.then, ActionMethodGet); err != nil {
				return derp.Wrap(err, "ghost.renderer.StepIfCondition.Get", "Error executing 'then' sub-steps", step.then)
			}
		}

		return nil
	}

	if len(step.otherwise) > 0 {
		if err := DoPipeline(renderer, buffer, step.otherwise, ActionMethodGet); err != nil {
			return derp.Wrap(err, "ghost.renderer.StepIfCondition.Get", "Error executing 'otherwise' sub-steps", step.then)
		}
	}

	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepIfCondition) Post(buffer io.Writer, renderer Renderer) error {

	if step.evaluateCondition(renderer) {
		if len(step.then) > 0 {
			if err := DoPipeline(renderer, buffer, step.then, ActionMethodPost); err != nil {
				return derp.Wrap(err, "ghost.renderer.StepIfCondition.Get", "Error executing 'then' sub-steps", step.then)
			}
		}
		return nil
	}

	if len(step.otherwise) > 0 {
		if err := DoPipeline(renderer, buffer, step.otherwise, ActionMethodPost); err != nil {
			return derp.Wrap(err, "ghost.renderer.StepIfCondition.Get", "Error executing 'otherwise' sub-steps", step.then)
		}
	}

	return nil
}

// evaluateCondition executes the conditional template and
func (step StepIfCondition) evaluateCondition(renderer Renderer) bool {
	result, _ := executeSingleTemplate(step.condition, renderer)
	return (strings.TrimSpace(result) == "true")
}
