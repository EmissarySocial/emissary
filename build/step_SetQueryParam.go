package build

import (
	"io"
	"text/template"
)

// StepSetQueryParam is an action-step that sets values to the request query string
type StepSetQueryParam struct {
	Values map[string]*template.Template
}

// Get displays a form where users can update stream data
func (step StepSetQueryParam) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.Do(builder)
}

// Post updates the stream with approved data from the request body.
func (step StepSetQueryParam) Post(builder Builder, _ io.Writer) PipelineBehavior {
	return step.Do(builder)
}

func (step StepSetQueryParam) Do(builder Builder) PipelineBehavior {
	query := builder.request().URL.Query()

	for key, value := range step.Values {
		queryValue := executeTemplate(value, builder)
		query.Set(key, queryValue)
	}

	builder.request().URL.RawQuery = query.Encode()
	return nil
}
