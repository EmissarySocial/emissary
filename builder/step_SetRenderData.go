package builder

import (
	"io"
	"text/template"
)

// StepSetRenderData represents an action-step that sets values to the request query string
type StepSetRenderData struct {
	Values map[string]*template.Template
}

// Get displays a form where users can update stream data
func (step StepSetRenderData) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.Do(builder)
}

// Post updates the stream with approved data from the request body.
func (step StepSetRenderData) Post(builder Builder, _ io.Writer) PipelineBehavior {
	return step.Do(builder)
}

func (step StepSetRenderData) Do(builder Builder) PipelineBehavior {
	for key, value := range step.Values {
		queryValue := executeTemplate(value, builder)
		builder.setString(key, queryValue)
	}

	return nil
}
