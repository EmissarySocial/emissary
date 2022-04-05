package render

import (
	"io"

	"github.com/benpate/derp"
	"github.com/whisperverse/whisperverse/model/step"
)

type StepError struct {
	Original step.Step
}

func (step StepError) Get(renderer Renderer, buffer io.Writer) error {
	return derp.NewInternalError("render.StepError", "Unrecognized Pipeline Step", "This should never happen", step.Original)
}

func (step StepError) Post(renderer Renderer, buffer io.Writer) error {
	return derp.NewInternalError("render.StepError", "Unrecognized Pipeline Step", "This should never happen", step.Original)
}
