package builder

import (
	"io"

	"github.com/EmissarySocial/emissary/model/step"
)

type Pipeline []step.Step

// Execute switches between GET and POST methods for this pipeline, based on the provided ActionMethod
func (pipeline Pipeline) Execute(factory Factory, builder Builder, buffer io.Writer, actionMethod ActionMethod) PipelineResult {

	if actionMethod == ActionMethodGet {
		return pipeline.Get(factory, builder, buffer)
	}

	return pipeline.Post(factory, builder, buffer)
}

// Get runs all of the pipeline steps using the GET method
func (pipeline Pipeline) Get(factory Factory, builder Builder, buffer io.Writer) PipelineResult {

	status := NewPipelineResult()

	// Execute all of the steps of the requested action
	for _, step := range pipeline {

		// Execute the step and collect the results in the pipeline status
		resultFn := ExecutableStep(step).Get(builder, buffer)

		if resultFn != nil {
			resultFn(&status)
		}

		if status.Halt {
			return status
		}
	}

	return status
}

// Post runs runs all of the pipeline steps using the POST method
func (pipeline Pipeline) Post(factory Factory, builder Builder, buffer io.Writer) PipelineResult {

	status := NewPipelineResult()

	// Execute all of the steps of the requested action
	for _, step := range pipeline {

		resultFn := ExecutableStep(step).Post(builder, buffer)

		if resultFn != nil {
			resultFn(&status)
		}

		if status.Halt {
			return status
		}
	}

	return status
}

func (pipeline Pipeline) IsEmpty() bool {
	return len(pipeline) == 0
}
