package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

type StepError struct {
	Original step.Step
}

func (step StepError) Get(renderer Renderer, buffer io.Writer) error {
	return derp.NewInternalError("render.StepError", "Unrecognized Pipeline Step", "This should never happen", renderer.ActionID(), renderer.Action(), renderer.Action().Steps, renderer.object(), step.Original)
}

func (step StepError) UseGlobalWrapper() bool {
	return true
}

func (step StepError) Post(renderer Renderer) error {
	return derp.NewInternalError("render.StepError", "Unrecognized Pipeline Step", "This should never happen", step.Original)
}
