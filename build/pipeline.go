package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model/step"
)

// Pipeline is an ordered list of Steps that build one action
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
		if resultFn := ExecutableStep(step).Get(builder, buffer); resultFn != nil {
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

		// Execute the step and collect the results in the pipeline status
		if resultFn := ExecutableStep(step).Post(builder, buffer); resultFn != nil {
			resultFn(&status)
		}

		if status.Halt {
			return status
		}
	}

	return status
}

// IsEmpty returns TRUE if this Pipeline contains no Steps
func (pipeline Pipeline) IsEmpty() bool {
	return len(pipeline) == 0
}

// Includes reports whether this Pipeline, or one nested in its steps, runs the named action through include
func (pipeline Pipeline) Includes(actionID string) bool {

	for _, current := range pipeline {

		if include, ok := current.(step.Include); ok && include.Action == actionID {
			return true
		}

		for _, nested := range nestedPipelines(current) {
			if nested.Includes(actionID) {
				return true
			}
		}
	}

	return false
}

// EditsAttachments reports whether this Pipeline, or one nested in its steps, holds an edit-attachments step for the field
func (pipeline Pipeline) EditsAttachments(fieldname string) bool {

	for _, current := range pipeline {

		if own, ok := current.(step.EditAttachments); ok && own.Fieldname == fieldname {
			return true
		}

		for _, nested := range nestedPipelines(current) {
			if nested.EditsAttachments(fieldname) {
				return true
			}
		}
	}

	return false
}

// nestedPipelines returns every Pipeline a step holds, which is how a step's own branches and
// sub-steps are reached without naming each step type
func nestedPipelines(current step.Step) []Pipeline {

	nested := step.Nested(current)
	result := make([]Pipeline, len(nested))

	for index, steps := range nested {
		result[index] = steps
	}

	return result
}
