package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepError is the Step that stands in for an unrecognized step name, and always fails
type StepError struct {
	Original step.Step
}

// Get renders this step during a GET request. Implements the Step interface.
func (step StepError) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return Halt().WithError(derp.Internal("build.StepError", "Unrecognized Pipeline Step", "This should never happen", builder.actionID(), builder.action(), builder.action().Steps, builder.object(), step.Original))
}

// Post applies this step during a POST request. Implements the Step interface.
func (step StepError) Post(builder Builder, _ io.Writer) PipelineBehavior {
	return Halt().WithError(derp.Internal("build.StepError", "Unrecognized Pipeline Step", "This should never happen", step.Original))
}
