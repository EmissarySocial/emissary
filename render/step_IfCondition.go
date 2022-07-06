package render

import (
	"bytes"
	"html/template"
	"io"
	"strings"

	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepIfCondition represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepIfCondition struct {
	Condition *template.Template
	Then      []step.Step
	Otherwise []step.Step
}

// Get displays a form where users can update stream data
func (step StepIfCondition) Get(renderer Renderer, buffer io.Writer) error {

	const location = "renderer.StepIfCondition.Get"

	factory := renderer.factory()

	if step.evaluateCondition(renderer) {
		if err := Pipeline(step.Then).Get(factory, renderer, buffer); err != nil {
			return derp.Wrap(err, location, "Error executing 'then' sub-steps")
		}

		return nil
	}

	if err := Pipeline(step.Otherwise).Get(factory, renderer, buffer); err != nil {
		return derp.Wrap(err, location, "Error executing 'otherwise' sub-steps")
	}

	return nil
}

func (step StepIfCondition) UseGlobalWrapper() bool {
	return useGlobalWrapper(step.Then) && useGlobalWrapper(step.Otherwise)
}

// Post updates the stream with approved data from the request body.
func (step StepIfCondition) Post(renderer Renderer) error {

	const location = "renderer.StepIfCondition.Post"

	factory := renderer.factory()

	if step.evaluateCondition(renderer) {
		if err := Pipeline(step.Then).Post(factory, renderer); err != nil {
			return derp.Wrap(err, location, "Error executing 'then' sub-steps")
		}

		return nil
	}

	if err := Pipeline(step.Otherwise).Post(factory, renderer); err != nil {
		return derp.Wrap(err, location, "Error executing 'otherwise' sub-steps")
	}

	return nil
}

// evaluateCondition executes the conditional template and
func (step StepIfCondition) evaluateCondition(renderer Renderer) bool {

	var result bytes.Buffer

	if err := step.Condition.Execute(&result, renderer); err != nil {
		return false
	}

	return (strings.TrimSpace(result.String()) == "true")
}
