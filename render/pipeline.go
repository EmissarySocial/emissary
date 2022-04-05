package render

import (
	"io"

	"github.com/benpate/derp"
	"github.com/whisperverse/whisperverse/model/step"
)

type Pipeline []step.Step

// Get runs all of the pipeline steps using the GET method
func (pipeline Pipeline) Get(factory Factory, renderer Renderer, buffer io.Writer) error {

	const location = "render.pipeline.Get"

	// Execute all of the steps of the requested action
	for _, step := range pipeline {

		// Fall through implies GET
		if err := step.Get(buffer, renderer); err != nil {
			return derp.Wrap(err, location, "Error GET-ing from step", step)
		}
	}

	return nil
}

// Post runs runs all of the pipeline steps using the POST method
func (pipeline Pipeline) Post(factory Factory, renderer Renderer, buffer io.Writer) error {

	const location = "render.pipeline.Post"

	// Execute all of the steps of the requested action
	for _, step := range pipeline {

		if err := step.Post(buffer, renderer); err != nil {
			return derp.Wrap(err, location, "Error POST-ing to step", step)
		}
	}

	return nil
}

func (pipeline Pipeline) IsEmpty() bool {
	return len(pipeline) == 0
}
