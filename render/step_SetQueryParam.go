package render

import (
	"io"

	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/maps"
)

// StepSetQueryParam represents an action-step that sets values to the request query string
type StepSetQueryParam struct {
	Values maps.Map
}

// Get displays a form where users can update stream data
func (step StepSetQueryParam) Get(renderer Renderer, buffer io.Writer) error {
	return step.Do(renderer)
}

func (step StepSetQueryParam) UseGlobalWrapper() bool {
	return true
}

// Post updates the stream with approved data from the request body.
func (step StepSetQueryParam) Post(renderer Renderer) error {
	return step.Do(renderer)
}

func (step StepSetQueryParam) Do(renderer Renderer) error {
	query := renderer.context().Request().URL.Query()

	for key, value := range step.Values {
		query.Set(key, convert.String(value))
	}

	renderer.context().Request().URL.RawQuery = query.Encode()
	return nil
}
