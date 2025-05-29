package step

import "github.com/benpate/rosetta/mapof"

// SetResponse is a Step that can create/update a response to the current model object
type SetResponse struct{}

// NewSetResponse returns a fully initialized SetResponse object
func NewSetResponse(stepInfo mapof.Any) (SetResponse, error) {

	return SetResponse{}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step SetResponse) Name() string {
	return "set-response"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step SetResponse) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step SetResponse) RequiredRoles() []string {
	return []string{}
}
