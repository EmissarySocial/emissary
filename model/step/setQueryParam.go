package step

import (
	"github.com/benpate/rosetta/maps"
)

// SetQueryParam represents an action-step that forwards the user to a new page.
type SetQueryParam struct {
	Values maps.Map
}

// NewSetQueryParam returns a fully initialized SetQueryParam object
func NewSetQueryParam(stepInfo maps.Map) (SetQueryParam, error) {

	stepInfo.DeletePath("step")

	return SetQueryParam{
		Values: stepInfo,
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step SetQueryParam) AmStep() {}
