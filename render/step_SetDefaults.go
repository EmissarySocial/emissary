package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/path"
)

// StepStreamDefaults sets a fixed set of values into a Stream
type StepStreamDefaults datatype.Map

func NewStepStreamDefaults(config datatype.Map) StepStreamDefaults {

	result := make(StepStreamDefaults)

	for key, value := range config {
		if key != "step" {
			result[key] = value
		}
	}

	return result
}

func (step StepStreamDefaults) Get(buffer io.Writer, renderer *Renderer) error {
	return nil
}

func (step StepStreamDefaults) Post(buffer io.Writer, renderer *Renderer) error {

	for key, value := range step {
		if err := path.Set(renderer.stream, key, value); err != nil {
			return derp.Wrap(err, "ghost.render.StepStreamDefaults.Post", "Error setting value")
		}
	}

	return nil
}
