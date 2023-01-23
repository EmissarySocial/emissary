package step

import "github.com/benpate/rosetta/mapof"

// SetQueryParam represents an action-step that forwards the user to a new page.
type SetQueryParam struct {
	Values mapof.Any
}

// NewSetQueryParam returns a fully initialized SetQueryParam object
func NewSetQueryParam(stepInfo mapof.Any) (SetQueryParam, error) {

	stepInfo.Remove("step")

	return SetQueryParam{
		Values: stepInfo,
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step SetQueryParam) AmStep() {}
