package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
)

// NewPipeline converts an array of maps into an array of steps
func NewPipeline(factory Factory, steps []datatype.Map) ([]Step, error) {

	const location = "render.NewPipeline"

	result := make([]Step, len(steps))

	for index := range steps {
		step, err := NewStep(factory, steps[index])

		if err != nil {
			return result, derp.Wrap(err, location, "Error parsing step", index, steps[index])
		}

		result[index] = step
	}

	return result, nil
}

// DoPipeline executes a series of RenderSteps on a particular Stream
func DoPipeline(renderer Renderer, buffer io.Writer, steps []datatype.Map, method ActionMethod) error {

	factory := renderer.factory()

	// Execute all of the steps of the requested action
	for _, stepInfo := range steps {

		step, err := NewStep(factory, stepInfo)

		if err != nil {
			return derp.Wrap(err, "whisper.render.DoPipeline", "Error initializing step", stepInfo)
		}

		if method == ActionMethodPost {
			err = step.Post(buffer, renderer)

		} else {
			err = step.Get(buffer, renderer)
		}

		if err != nil {
			return derp.Wrap(err, "whisper.render.DoPipeline", "Error executing step", stepInfo)
		}
	}

	return nil
}
