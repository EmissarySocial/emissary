package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

type StepError struct {
	Original step.Step
}

func (step StepError) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return Halt().WithError(derp.NewInternalError("build.StepError", "Unrecognized Pipeline Step", "This should never happen", builder.ActionID(), builder.Action(), builder.Action().Steps, builder.object(), step.Original))
}

func (step StepError) Post(builder Builder, _ io.Writer) PipelineBehavior {
	return Halt().WithError(derp.NewInternalError("build.StepError", "Unrecognized Pipeline Step", "This should never happen", step.Original))
}
