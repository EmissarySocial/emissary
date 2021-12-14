package render

import (
	"io"

	"github.com/benpate/datatype"
)

// StepStreamShare represents an action that can edit a top-level folder in the Domain
type StepStreamShare struct {
}

// NewStepStreamShare returns a fully parsed StepStreamShare object
func NewStepStreamShare(config datatype.Map) StepStreamShare {

	return StepStreamShare{}
}

func (step StepStreamShare) Get(buffer io.Writer, renderer Renderer) error {
	return nil
}

func (step StepStreamShare) Post(buffer io.Writer, renderer Renderer) error {
	return nil
}
