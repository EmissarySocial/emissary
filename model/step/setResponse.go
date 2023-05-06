package step

import "github.com/benpate/rosetta/mapof"

// SetResponse represents an action-step that can create/update a response to the current model object
type SetResponse struct{}

// NewSetResponse returns a fully initialized SetResponse object
func NewSetResponse(stepInfo mapof.Any) (SetResponse, error) {

	return SetResponse{}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step SetResponse) AmStep() {}
