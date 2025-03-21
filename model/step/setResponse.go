package step

import "github.com/benpate/rosetta/mapof"

// SetResponse is a Step that can create/update a response to the current model object
type SetResponse struct{}

// NewSetResponse returns a fully initialized SetResponse object
func NewSetResponse(stepInfo mapof.Any) (SetResponse, error) {

	return SetResponse{}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step SetResponse) AmStep() {}
