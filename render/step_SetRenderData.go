package render

import (
	"io"
	"text/template"
)

// StepSetRenderData represents an action-step that sets values to the request query string
type StepSetRenderData struct {
	Values map[string]*template.Template
}

// Get displays a form where users can update stream data
func (step StepSetRenderData) Get(renderer Renderer, buffer io.Writer) PipelineBehavior {
	return step.Do(renderer)
}

// Post updates the stream with approved data from the request body.
func (step StepSetRenderData) Post(renderer Renderer, _ io.Writer) PipelineBehavior {
	return step.Do(renderer)
}

func (step StepSetRenderData) Do(renderer Renderer) PipelineBehavior {
	for key, value := range step.Values {
		queryValue := executeTemplate(value, renderer)
		renderer.setString(key, queryValue)
	}

	return nil
}
